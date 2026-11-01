// Generated JavaScript: edit this TypeScript source and run npm run build.
export function splitPages(text: string) { return String(text || '').split(/\r?\n---PAGE---\r?\n/); }
export function readDocument(file: Blob): Promise<string> {
  return new Promise<string>((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result).split(',')[1]);
    reader.onerror = () => reject(new Error('Não foi possível ler o arquivo'));
    reader.readAsDataURL(file);
  });
}

export function joinPages(pages: string[]) { return pages.join("\n---PAGE---\n"); }
let activeOriginalUrl: string | undefined;
export function originalUrl(bytes: Uint8Array | ArrayBuffer | number[], name: string) {
  const data = bytes instanceof Uint8Array ? Uint8Array.from(bytes) : new Uint8Array(bytes);
  if (!data.length) throw new Error('O servidor não retornou os bytes do original');
  const extension = String(name).toLowerCase().split('.').pop();
  const mime = ({pdf: 'application/pdf', png: 'image/png', jpg: 'image/jpeg', jpeg: 'image/jpeg', webp: 'image/webp'} as Record<string, string>)[extension || ''];
  if (!mime) throw new Error('Formato não suportado');
  if (activeOriginalUrl) URL.revokeObjectURL(activeOriginalUrl);
  activeOriginalUrl = URL.createObjectURL(new Blob([data], {type: mime}));
  return activeOriginalUrl;
}
export function localDate() {
  const date = new Date();
  return [date.getFullYear(), String(date.getMonth()+1).padStart(2,'0'), String(date.getDate()).padStart(2,'0')].join('-');
}
export function bindPrint(button: HTMLElement, tasks: {task_id: string; title: string; done: boolean}[], date: string, container: HTMLElement) {
  button.addEventListener('click', () => {
    const popup = window.open('', '_blank');
    if (!popup) return;
    popup.opener = null;
    popup.document.title = 'Paperwork';
    const style = popup.document.createElement('style');
    style.textContent = '@page{size:A4;margin:18mm}body{font:14pt sans-serif;line-height:2}pre{white-space:pre-wrap;font:inherit}';
    popup.document.head.appendChild(style);
    const content = popup.document.createElement('pre');
    const current = new Map([...container.querySelectorAll<HTMLInputElement>('input[data-paperwork-task]')].map(input=>[input.dataset.paperworkTask,input.checked]));
    content.textContent = '// '+date+'\n'+tasks.filter(task=>!(current.has(task.task_id) ? current.get(task.task_id) : task.done)).map(task=>'□ '+task.task_id+' '+task.title).join('\n')+'\n//end';
    popup.document.body.appendChild(content);
    popup.focus(); popup.print();
  });
}
export function watchEdits(element: HTMLElement) {
  element.dataset.paperworkDirty = 'false';
  element.addEventListener('input', () => { element.dataset.paperworkDirty = 'true'; });
}
export function markReviewed(element: HTMLElement) { element.dataset.paperworkDirty = 'false'; }
export function hasUnreviewedEdits(element: HTMLElement) { return element.dataset.paperworkDirty === 'true'; }
export function markDirty(element: HTMLElement) { element.dataset.paperworkDirty = 'true'; }

let activeCamera: ReturnType<typeof cameraPanel> | undefined;
export function cameraPanel(): {root: HTMLDivElement; close: () => void; photo: () => Promise<{name: string; data: string}>} {
  activeCamera?.close();
  const root = document.createElement('div');
  const video = document.createElement('video');
  video.autoplay = true; video.muted = true; video.playsInline = true;
  video.setAttribute('playsinline', ''); video.style.cssText = 'max-width:100%;max-height:45vh';
  const image = document.createElement('img');
  image.alt = 'Foto da página antes de importar'; image.style.cssText = 'max-width:100%;max-height:45vh';
  const status = document.createElement('p'); status.setAttribute('role', 'status');
  const start = document.createElement('button'); start.textContent = 'Ligar câmera';
  const take = document.createElement('button'); take.textContent = 'Fotografar'; take.disabled = true;
  const native = document.createElement('input'); native.type = 'file'; native.accept = 'image/jpeg,image/png,image/webp';
  native.setAttribute('capture', 'environment'); native.setAttribute('aria-label', 'Câmera nativa ou foto do aparelho');
  const help = document.createElement('p'); help.textContent = 'Alternativa: use o seletor abaixo para abrir a câmera nativa do celular ou escolher uma foto.';
  let stream: MediaStream | null = null, selected: File | undefined, closed = false, preview: string | undefined;
  const stop = () => { stream?.getTracks().forEach(track => track.stop()); stream = null; video.srcObject = null; take.disabled = true; };
  const show = (file: File) => {
    if (preview) URL.revokeObjectURL(preview);
    selected = file; preview = URL.createObjectURL(file); image.src = preview;
    status.textContent = 'Foto pronta. Confira e clique em Importar foto.';
  };
  start.addEventListener('click', async () => {
    stop();
    try {
      if (!navigator.mediaDevices?.getUserMedia) throw new Error('Use HTTPS e um navegador com suporte à câmera, ou o seletor abaixo.');
      const incoming = await navigator.mediaDevices.getUserMedia({video: {facingMode: {ideal: 'environment'}}, audio: false});
      if (closed) {incoming.getTracks().forEach(track => track.stop()); return;}
      stream = incoming; video.srcObject = stream; await video.play(); take.disabled = false;
      status.textContent = 'Enquadre a página e clique em Fotografar.';
    } catch {stop(); status.textContent = 'Não foi possível abrir a câmera. Confira a permissão do navegador ou use o seletor abaixo.';}
  });
  take.addEventListener('click', () => {
    if (!video.videoWidth || !video.videoHeight) { status.textContent = 'Aguarde a câmera ficar pronta.'; return; }
    const canvas = document.createElement('canvas');
    // Keep native resolution; no crop, OCR or automatic image changes.
    canvas.width = video.videoWidth; canvas.height = video.videoHeight;
    const context = canvas.getContext('2d');
    if (!context) { status.textContent = 'Não foi possível capturar a imagem.'; return; }
    context.drawImage(video, 0, 0);
    canvas.toBlob(blob => { if (!closed && blob) {show(new File([blob], 'pagina-'+Date.now()+'.jpg', {type:'image/jpeg'})); stop();} }, 'image/jpeg', .95);
  });
  native.addEventListener('change', () => { if (native.files?.[0]) { stop(); show(native.files[0]); } });
  const onHidden = () => {if (document.hidden) stop();};
  const onPageHide = () => close();
  document.addEventListener('visibilitychange', onHidden); window.addEventListener('pagehide', onPageHide);
  const observer = new MutationObserver(() => { if (!root.isConnected) close(); });
  observer.observe(document.body, {childList:true, subtree:true});
  function close() {
    if (closed) return; closed = true; stop(); observer.disconnect();
    document.removeEventListener('visibilitychange', onHidden); window.removeEventListener('pagehide', onPageHide);
    if (preview) URL.revokeObjectURL(preview);
  }
  root.append(video, start, take, image, help, native, status);
  const result = {root, close, photo: async () => {
    if (!selected) throw new Error('Tire ou selecione uma foto primeiro.');
    if (selected.size > 18000000) throw new Error('A foto excede 18 MB.');
    return {name:selected.name, data:await readDocument(selected)};
  }};
  activeCamera = result; return result;
}
