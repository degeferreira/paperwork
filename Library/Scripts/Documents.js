export function splitPages(text) { return String(text || '').split(/\r?\n---PAGE---\r?\n/); }
export function readDocument(file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result).split(',')[1]);
    reader.onerror = () => reject(new Error('Não foi possível ler o arquivo'));
    reader.readAsDataURL(file);
  });
}

export function joinPages(pages) { return pages.join("\n---PAGE---\n"); }
let activeOriginalUrl;
export function originalUrl(bytes, name) {
  const data = bytes instanceof Uint8Array ? bytes : new Uint8Array(bytes);
  if (!data.length) throw new Error('O servidor não retornou os bytes do original');
  const extension = String(name).toLowerCase().split('.').pop();
  const mime = {pdf: 'application/pdf', png: 'image/png', jpg: 'image/jpeg', jpeg: 'image/jpeg', webp: 'image/webp'}[extension];
  if (!mime) throw new Error('Formato não suportado');
  if (activeOriginalUrl) URL.revokeObjectURL(activeOriginalUrl);
  activeOriginalUrl = URL.createObjectURL(new Blob([data], {type: mime}));
  return activeOriginalUrl;
}
export function localDate() {
  const date = new Date();
  return [date.getFullYear(), String(date.getMonth()+1).padStart(2,'0'), String(date.getDate()).padStart(2,'0')].join('-');
}
export function bindPrint(button, tasks, date, container) {
  button.addEventListener('click', () => {
    const popup = window.open('', '_blank');
    if (!popup) return;
    popup.opener = null;
    popup.document.title = 'Paperwork';
    const style = popup.document.createElement('style');
    style.textContent = '@page{size:A4;margin:18mm}body{font:14pt sans-serif;line-height:2}pre{white-space:pre-wrap;font:inherit}';
    popup.document.head.appendChild(style);
    const content = popup.document.createElement('pre');
    const current = new Map([...container.querySelectorAll('input[data-paperwork-task]')].map(input=>[input.dataset.paperworkTask,input.checked]));
    content.textContent = '// '+date+'\n'+tasks.filter(task=>!(current.has(task.task_id) ? current.get(task.task_id) : task.done)).map(task=>'□ '+task.task_id+' '+task.title).join('\n')+'\n//end';
    popup.document.body.appendChild(content);
    popup.focus(); popup.print();
  });
}
export function watchEdits(element) {
  element.dataset.paperworkDirty = 'false';
  element.addEventListener('input', () => { element.dataset.paperworkDirty = 'true'; });
}
export function markReviewed(element) { element.dataset.paperworkDirty = 'false'; }
export function hasUnreviewedEdits(element) { return element.dataset.paperworkDirty === 'true'; }
export function markDirty(element) { element.dataset.paperworkDirty = 'true'; }
