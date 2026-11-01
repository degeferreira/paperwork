const vm=require('node:vm'),fs=require('node:fs'),path=require('node:path'),assert=require('node:assert/strict');
const root=path.join(__dirname,'../zotero');
let clipboard='',preferences={},prompts=0,saves=0;
const elements=new Map(),listeners=new Map();
const menu={appendChild(e){elements.set(e.id,e);}};
elements.set('zotero-itemmenu',menu);
function item(key,title,extra='DOI: 10.1/test') {return {key,libraryID:1,deleted:false,values:{title,date:'2024',extra},
 isRegularItem:()=>true,getField(name){return this.values[name] || '';},getCreators:()=>[{lastName:'Silva'}],
 setField(name,value){assert.equal(name,'extra');this.values[name]=value;},async saveTx(){saves++;}};}
const a=item('ABCD1234','Memória social'),b=item('EFGH5678','Memória coletiva'),c=item('IJKL1234','Memória outra');
let selection=[a,b];
const win={ZoteroPane:{getSelectedItems:()=>selection},document:{getElementById:id=>elements.get(id),createXULElement:()=>({id:'',setAttribute(){},addEventListener(type,fn){listeners.set(this.id,fn);},remove(){elements.delete(this.id);}})}};
const scope=vm.createContext({console,JSON,Object,Promise,Services:{prompt:{confirm(){prompts++;return true;},alert(w,t,message){throw Error(message);}},scriptloader:{loadSubScript(){vm.runInContext(fs.readFileSync(root+'/aliases.js','utf8'),scope);}}},
 Cc:{'@mozilla.org/widget/clipboardhelper;1':{getService:()=>({copyString:text=>clipboard=text})}},Ci:{nsIClipboardHelper:{}},
 Zotero:{Prefs:{get:k=>preferences[k],set:(k,v)=>preferences[k]=v},getMainWindows:()=>[win],logError:e=>{throw e;},
 URI:{getItemPath:i=>'users/123/items/'+i.key},Libraries:{getAll:()=>[{libraryID:1}],get:()=>({editable:true})},Items:{getAll:async()=>[a,b,c]}}});
vm.runInContext(fs.readFileSync(root+'/bootstrap.js','utf8'),scope);
(async()=>{
 await scope.startup({rootURI:'file:///fixture/'});assert(elements.has('paperwork-create-alias'));
 await scope.create(win);
 assert.equal(saves,2);assert.equal(prompts,1);assert.equal(clipboard,'Ƶ Silva24-Mem\nƵ Silva24-Mem-2');
 assert(a.values.extra.startsWith('DOI: 10.1/test\n'));assert.equal(a.values.title,'Memória social');
 await scope.create(win);assert.equal(saves,2);assert.equal(prompts,1);
 a.values.title='Novo título';await scope.create(win);assert(clipboard.includes('Ƶ Silva24-Mem\n'));
 await scope.exportIndex(win);const index=JSON.parse(clipboard);assert.equal(index.references[0].library_id,123);assert.equal(index.references[1].item_key,'EFGH5678');
 // Deleting an alias from the item does not free its previous reservation.
 a.values.extra='';selection=[c];await scope.create(win);assert.equal(clipboard,'Ƶ Silva24-Mem-3');
 await scope.shutdown();assert(!elements.has('paperwork-create-alias'));assert(!elements.has('paperwork-export-index'));
 console.log('Zotero addon contracts: bootstrap lifecycle, explicit Extra writes, stable IDs, collision/tombstone protection, index export.');
})().catch(error=>{console.error(error);process.exitCode=1;});
