/* Run with playwright-core installed and CHROMIUM_PATH pointing to a browser. */
const {chromium} = require('playwright-core');
const http = require('node:http');
const fs = require('node:fs');
const path = require('node:path');
const assert = require('node:assert/strict');
(async () => {
 const server = http.createServer((req,res) => {
   if (req.url==='/Documents.js') {res.setHeader('Content-Type','text/javascript');res.end(fs.readFileSync(path.join(__dirname,'../Library/Scripts/Documents.js')));}
   else res.end('<!doctype html><html><body><script type="module">import {cameraPanel} from "/Documents.js";window.makeCamera=()=>{window.camera=cameraPanel();document.body.appendChild(camera.root);};window.ready=true;</script></body></html>');
 });
 await new Promise(resolve=>server.listen(0,'127.0.0.1',resolve));
 let browser;
 try {
  const origin='http://127.0.0.1:'+server.address().port;
  browser=await chromium.launch({executablePath:process.env.CHROMIUM_PATH || '/usr/bin/chromium',headless:true,args:['--no-sandbox','--use-fake-device-for-media-stream','--use-fake-ui-for-media-stream','--disable-dev-shm-usage']});
  const context=await browser.newContext({permissions:['camera']});
  const page=await context.newPage();await page.goto(origin);await page.waitForFunction(()=>window.ready);
  await page.evaluate(()=>{
   const original=navigator.mediaDevices.getUserMedia.bind(navigator.mediaDevices);
   navigator.mediaDevices.getUserMedia=async options=>{const s=await original(options);window.lastTracks=s.getTracks();return s;};
   window.makeCamera();
  });
  await page.getByText('Ligar câmera',{exact:true}).click();
  await page.waitForFunction(()=>document.querySelector('video').videoWidth>0 && ![...document.querySelectorAll('button')].find(b=>b.textContent==='Fotografar').disabled);
  await page.getByText('Fotografar',{exact:true}).click();await page.getByRole('status').filter({hasText:'Foto pronta'}).waitFor();
  const photo=await page.evaluate(()=>window.camera.photo());
  assert.equal(Buffer.from(photo.data,'base64').subarray(0,3).toString('hex'),'ffd8ff');
  assert(await page.evaluate(()=>window.lastTracks.every(t=>t.readyState==='ended')));
  await page.evaluate(()=>{window.camera.close();window.camera.root.remove();window.makeCamera();});
  const original=Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/l9sAAAAASUVORK5CYII=','base64');
  await page.locator('input[type=file]').setInputFiles({name:'original.png',mimeType:'image/png',buffer:original});
  const uploaded=await page.evaluate(()=>window.camera.photo());assert.deepEqual(Buffer.from(uploaded.data,'base64'),original);
  await page.evaluate(()=>{window.camera.close();window.camera.root.remove();window.makeCamera();});
  await page.getByText('Ligar câmera',{exact:true}).click();await page.waitForFunction(()=>document.querySelector('video').videoWidth>0);
  await page.evaluate(()=>window.camera.root.remove());await page.waitForFunction(()=>window.lastTracks.every(t=>t.readyState==='ended'));
  console.log('Chromium: captured JPEG, stopped camera tracks, preserved uploaded bytes and cleaned up removed panel.');
 } finally {await browser?.close();server.close();}
})().catch(error=>{console.error(error);process.exitCode=1;});
