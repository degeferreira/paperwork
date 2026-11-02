/* Run with playwright-core installed and CHROMIUM_PATH pointing to a browser. */
const {chromium} = require('playwright-core');
const http = require('node:http');
const fs = require('node:fs');
const path = require('node:path');
const assert = require('node:assert/strict');
(async () => {
 const server = http.createServer((req,res) => {
   if (req.url==='/Documents.js') {res.setHeader('Content-Type','text/javascript');res.end(fs.readFileSync(path.join(__dirname,'../Library/Scripts/Documents.js')));}
   else res.end('<!doctype html><html><body><script type="module">import {cameraPanel} from "/Documents.js";window.makeCamera=()=>{window.camera=cameraPanel();document.body.appendChild(camera.root);window.confirmPhoto=document.createElement("button");window.confirmPhoto.textContent="Usar esta foto";camera.root.appendChild(confirmPhoto);camera.bindConfirmation(confirmPhoto);};window.ready=true;</script></body></html>');
 });
 await new Promise(resolve=>server.listen(0,'127.0.0.1',resolve));
 let browser;
 try {
  const origin='http://127.0.0.1:'+server.address().port;
  browser=await chromium.launch({executablePath:process.env.CHROMIUM_PATH || '/usr/bin/chromium',headless:true,args:['--disable-gpu','--use-angle=swiftshader','--no-sandbox','--no-proxy-server','--use-fake-device-for-media-stream','--use-fake-ui-for-media-stream','--disable-dev-shm-usage']});
  const context=await browser.newContext({permissions:['camera']});
  const page=await context.newPage();await page.goto(origin,{waitUntil:"domcontentloaded",timeout:60000});await page.waitForFunction(()=>window.ready);
  await page.setViewportSize({width:390,height:844});
  await page.evaluate(()=>{
   const original=navigator.mediaDevices.getUserMedia.bind(navigator.mediaDevices);
   navigator.mediaDevices.getUserMedia=async options=>{const s=await original(options);window.lastTracks=s.getTracks();return s;};
   window.makeCamera();
  });
  assert(await page.getByRole('button',{name:'Usar esta foto',exact:true}).isDisabled());
  const controls=await page.locator('button:visible').evaluateAll(buttons=>buttons.map(b=>({height:b.getBoundingClientRect().height,width:b.getBoundingClientRect().width})));assert(controls.every(b=>b.height>=48 && b.width>=48));
  await page.getByRole('button',{name:'Ativar câmera',exact:true}).click();
  await page.waitForFunction(()=>document.querySelector('video').videoWidth>0 && ![...document.querySelectorAll('button')].find(b=>b.getAttribute('aria-label')==='Fotografar').disabled);
  await page.getByRole('button',{name:'Fotografar',exact:true}).click();await page.getByRole('status').filter({hasText:'Foto pronta'}).waitFor();
  await page.getByRole('button',{name:'Tirar outra foto',exact:true}).waitFor();
  assert(await page.locator('video').isHidden());
  assert(await page.getByRole('button',{name:'Usar esta foto',exact:true}).isEnabled());
  const photo=await page.evaluate(()=>window.camera.photo());
  assert.equal(Buffer.from(photo.data,'base64').subarray(0,3).toString('hex'),'ffd8ff');
  assert(await page.evaluate(()=>window.lastTracks.every(t=>t.readyState==='ended')));
  await page.evaluate(()=>{window.camera.close();window.camera.root.remove();window.makeCamera();});
  const original=Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/l9sAAAAASUVORK5CYII=','base64');
  await page.locator('input[type=file]').setInputFiles({name:'original.png',mimeType:'image/png',buffer:original});
  const uploaded=await page.evaluate(()=>window.camera.photo());assert.deepEqual(Buffer.from(uploaded.data,'base64'),original);
  await page.evaluate(()=>{window.camera.close();window.camera.root.remove();window.makeCamera();});
  await page.emulateMedia({colorScheme:'dark'});
  await page.keyboard.press('Tab');
  await page.getByRole('button',{name:'Ativar câmera',exact:true}).focus();
  assert.equal(await page.getByRole('button',{name:'Ativar câmera',exact:true}).evaluate(b=>getComputedStyle(b).outlineStyle),'solid');
  await page.getByRole('button',{name:'Ativar câmera',exact:true}).click();await page.waitForFunction(()=>document.querySelector('video').videoWidth>0);
  await page.evaluate(()=>window.camera.root.remove());await page.waitForFunction(()=>window.lastTracks.every(t=>t.readyState==='ended'));
  await page.evaluate(()=>{window.makeCamera();window.savedMedia=navigator.mediaDevices.getUserMedia;navigator.mediaDevices.getUserMedia=async()=>{throw new DOMException('Denied','NotAllowedError')};});
  await page.getByRole('button',{name:'Ativar câmera',exact:true}).click();
  await page.getByRole('status').filter({hasText:'Câmera indisponível'}).waitFor();
  assert(await page.getByRole('button',{name:'Usar esta foto',exact:true}).isDisabled());
  assert(await page.getByRole('button',{name:'Usar foto do aparelho',exact:true}).isEnabled());
  await page.evaluate(()=>{window.camera.close();window.camera.root.remove();navigator.mediaDevices.getUserMedia=window.savedMedia;});
  console.log('Chromium: captured JPEG, stopped camera tracks, preserved uploaded bytes and cleaned up removed panel.');
 } finally {await browser?.close();server.close();}
})().catch(error=>{console.error(error);process.exitCode=1;});
