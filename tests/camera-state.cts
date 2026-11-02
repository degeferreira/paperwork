// State/DOM regression checks; browser layout is verified separately.
const {parseHTML}=require('linkedom');
const assert=require('node:assert/strict');
const path=require('node:path');
const {pathToFileURL}=require('node:url');
const {setTimeout:delay}=require('node:timers/promises');
(async()=>{
 const {window}=parseHTML('<!doctype html><html><body></body></html>');
 Object.assign(globalThis,{window,document:window.document,MutationObserver:window.MutationObserver,Event:window.Event});
 Object.defineProperty(document,'hidden',{configurable:true,value:false});
 class Reader{readAsDataURL(blob){blob.arrayBuffer().then(bytes=>{this.result='data:'+blob.type+';base64,'+Buffer.from(bytes).toString('base64');this.onload?.();}).catch(()=>this.onerror?.());}}
 globalThis.FileReader=Reader;
 const frame=Buffer.from([255,216,255,224,1,2,3]);
 const create=document.createElement.bind(document);
 document.createElement=(name,...args)=>{const element=create(name,...args);if(name==='video'){element.play=async()=>{element.videoWidth=640;element.videoHeight=480;};}if(name==='canvas'){element.getContext=()=>({drawImage(){}});element.toBlob=callback=>queueMicrotask(()=>callback(new Blob([frame],{type:'image/jpeg'})));}return element;};
 const tracks=[];const stream=()=>{const track={readyState:'live',stop(){this.readyState='ended'}};tracks.push(track);return{getTracks:()=>[track]}};
 Object.defineProperty(globalThis,'navigator',{configurable:true,value:{mediaDevices:{getUserMedia:async()=>stream()}}});
 const {cameraPanel,actionButton}=await import(pathToFileURL(path.join(__dirname,'../Library/Scripts/Documents.js')).href);
 const button=(panel,label)=>[...panel.root.querySelectorAll('button')].find(b=>b.getAttribute('aria-label')===label);
 const mount=async()=>{const panel=cameraPanel();const confirm=actionButton('Usar esta foto','check','primary');panel.root.append(confirm);panel.bindConfirmation(confirm);document.body.append(panel.root);await delay(0);return{panel,confirm}};
 let {panel,confirm}=await mount();assert(confirm.disabled);
 button(panel,'Ativar câmera').click();await delay(0);assert.equal(button(panel,'Fotografar').disabled,false);
 button(panel,'Fotografar').click();await delay(0);assert.equal(confirm.disabled,false,'photo must enable native confirmation button');assert.equal(tracks[0].readyState,'ended');assert.equal(panel.root.querySelector('video').hidden,true);
 assert.deepEqual(Buffer.from((await panel.photo()).data,'base64'),frame);
 button(panel,'Tirar outra foto').click();assert(confirm.disabled,'retake must clear previous confirmation');await delay(0);panel.close();assert(tracks.every(t=>t.readyState==='ended'));panel.root.remove();
 ({panel,confirm}=await mount());const original=Buffer.from([137,80,78,71,13,10,26,10]);const input=panel.root.querySelector('input[type=file]');Object.defineProperty(input,'files',{configurable:true,value:[new File([original],'original.png',{type:'image/png'})]});input.dispatchEvent(new window.Event('change'));assert.equal(confirm.disabled,false);assert.deepEqual(Buffer.from((await panel.photo()).data,'base64'),original);panel.close();panel.root.remove();
 navigator.mediaDevices.getUserMedia=async()=>{throw new DOMException('Denied','NotAllowedError')};({panel,confirm}=await mount());button(panel,'Ativar câmera').click();await delay(0);assert(confirm.disabled);assert(panel.root.querySelector('[role=status]').textContent.includes('Câmera indisponível'));assert.equal(button(panel,'Usar foto do aparelho').disabled,false);panel.close();panel.root.remove();
 let resolve; navigator.mediaDevices.getUserMedia=()=>new Promise(done=>resolve=done);({panel,confirm}=await mount());button(panel,'Ativar câmera').click();panel.close();const incoming=stream();resolve(incoming);await delay(0);assert(incoming.getTracks().every(t=>t.readyState==='ended'),'late permission must not reopen camera');assert(confirm.disabled);panel.root.remove();
 console.log('Camera DOM: capture/confirmation, retake, original bytes, denied and late permissions passed.');
})().catch(error=>{console.error(error);process.exitCode=1});
