// Generated JavaScript: edit this TypeScript source and run npm run build.
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
    const data = bytes instanceof Uint8Array ? Uint8Array.from(bytes) : new Uint8Array(bytes);
    if (!data.length)
        throw new Error('O servidor não retornou os bytes do original');
    const extension = String(name).toLowerCase().split('.').pop();
    const mime = { pdf: 'application/pdf', png: 'image/png', jpg: 'image/jpeg', jpeg: 'image/jpeg', webp: 'image/webp' }[extension || ''];
    if (!mime)
        throw new Error('Formato não suportado');
    if (activeOriginalUrl)
        URL.revokeObjectURL(activeOriginalUrl);
    activeOriginalUrl = URL.createObjectURL(new Blob([data], { type: mime }));
    return activeOriginalUrl;
}
export function localDate() {
    const date = new Date();
    return [date.getFullYear(), String(date.getMonth() + 1).padStart(2, '0'), String(date.getDate()).padStart(2, '0')].join('-');
}
export function bindPrint(button, tasks, date, container) {
    button.addEventListener('click', () => {
        const popup = window.open('', '_blank');
        if (!popup)
            return;
        popup.opener = null;
        popup.document.title = 'Paperwork';
        const style = popup.document.createElement('style');
        style.textContent = '@page{size:A4;margin:18mm}body{font:14pt sans-serif;line-height:2}pre{white-space:pre-wrap;font:inherit}';
        popup.document.head.appendChild(style);
        const content = popup.document.createElement('pre');
        const current = new Map([...container.querySelectorAll('input[data-paperwork-task]')].map(input => [input.dataset.paperworkTask, input.checked]));
        content.textContent = '// ' + date + '\n' + tasks.filter(task => !(current.has(task.task_id) ? current.get(task.task_id) : task.done)).map(task => '□ ' + task.task_id + ' ' + task.title).join('\n') + '\n//end';
        popup.document.body.appendChild(content);
        popup.focus();
        popup.print();
    });
}
export function watchEdits(element) {
    element.dataset.paperworkDirty = 'false';
    element.addEventListener('input', () => { element.dataset.paperworkDirty = 'true'; });
}
export function markReviewed(element) { element.dataset.paperworkDirty = 'false'; }
export function hasUnreviewedEdits(element) { return element.dataset.paperworkDirty === 'true'; }
export function markDirty(element) { element.dataset.paperworkDirty = 'true'; }
const panelCSS = `
.pw-ui{color-scheme:light dark;--pw-accent:#36594e;--pw-on-accent:#fff;--pw-border:#b5c3bc;--pw-muted:#53635b;--pw-soft:#f0f4ef;font:16px/1.5 system-ui,-apple-system,"Segoe UI",sans-serif;color:CanvasText;background:Canvas;box-sizing:border-box;isolation:isolate}
.pw-ui *{box-sizing:border-box}.pw-ui h2{font:650 clamp(23px,4vw,30px)/1.2 system-ui;margin:0 0 8px}.pw-ui h3{font:650 18px/1.3 system-ui;margin:24px 0 12px}.pw-ui p{margin:8px 0 16px}.pw-ui button,.pw-ui .pw-file-label{font:600 16px/1.35 system-ui;min-height:48px;min-width:48px;padding:12px 18px;border:1px solid var(--pw-border);border-radius:12px;background:Canvas;color:CanvasText;cursor:pointer;touch-action:manipulation;white-space:normal;overflow-wrap:anywhere}
.pw-ui > button{margin:4px 8px 4px 0;max-width:100%}.pw-ui button:hover:not(:disabled){background:var(--pw-soft)}.pw-ui button:disabled{opacity:.55;cursor:default}.pw-ui :is(button,input,select,textarea,a):focus-visible{outline:3px solid var(--pw-accent);outline-offset:3px}.pw-ui input:not([type=checkbox]):not([type=file]),.pw-ui select,.pw-ui textarea{font:inherit;min-height:48px;max-width:100%;border:1px solid var(--pw-border);border-radius:10px;padding:10px;background:Canvas;color:CanvasText}.pw-ui input[type=checkbox]{width:24px;height:24px;flex-shrink:0}.pw-ui label{min-height:48px;display:flex;gap:10px;align-items:center}
.pw-ui .pw-header{display:flex;align-items:flex-start;justify-content:space-between;gap:16px;margin-bottom:20px}.pw-ui .pw-header > div{min-width:0;flex:1}.pw-ui .pw-header .pw-action{width:auto;flex:0 0 auto}.pw-ui .pw-eyebrow{font-size:12px;font-weight:750;letter-spacing:.13em;color:var(--pw-muted);margin:0 0 8px}.pw-ui .pw-subtitle,.pw-ui .pw-hint{color:var(--pw-muted);font-size:15px}.pw-ui .pw-actions{display:grid;grid-template-columns:repeat(auto-fit,minmax(min(100%,240px),1fr));gap:12px;margin:16px 0}.pw-ui .pw-action{display:flex;align-items:center;gap:14px;text-align:left;width:100%}.pw-ui .pw-action svg{width:24px;height:24px;flex-shrink:0}.pw-ui .pw-action-copy{display:flex;flex-direction:column;gap:4px}.pw-ui .pw-action small{font-size:14px;font-weight:400;line-height:1.45}.pw-ui .pw-action:has(small){min-height:100px;padding:20px}.pw-ui .pw-primary{background:var(--pw-accent);color:var(--pw-on-accent);border-color:var(--pw-accent)}.pw-ui .pw-primary:hover:not(:disabled){background:var(--pw-accent);filter:brightness(1.08)}.pw-ui .pw-quiet{border-color:transparent;background:transparent}.pw-ui .pw-document{margin:0 0 10px}.pw-ui .pw-empty{padding:24px;border:1px dashed var(--pw-border);border-radius:14px;color:var(--pw-muted);text-align:center}.pw-ui [role=status]{font-size:15px;overflow-wrap:anywhere}.pw-ui [role=status]:empty{display:none}.pw-ui .pw-steps{display:flex;gap:8px;flex-wrap:wrap;padding:0;list-style:none;margin:0 0 24px;font-size:13px;color:var(--pw-muted)}.pw-ui .pw-steps li{padding:5px 10px;border-radius:20px;background:var(--pw-soft)}.pw-ui .pw-steps [aria-current=step]{background:var(--pw-accent);color:var(--pw-on-accent);font-weight:650}
.pw-ui .pw-camera-stage{display:flex;align-items:center;justify-content:center;min-height:200px;background:var(--pw-soft);border:1px solid var(--pw-border);border-radius:16px;overflow:hidden;margin:16px 0}.pw-ui .pw-camera-stage :is(video,img){display:block;max-width:100%;max-height:45vh;object-fit:contain}.pw-ui .pw-camera-stage [hidden],.pw-ui [hidden]{display:none!important}.pw-ui .pw-camera-placeholder{padding:28px;text-align:center;color:var(--pw-muted)}.pw-ui .pw-camera-placeholder svg{width:44px;height:44px;margin-bottom:8px}.pw-ui .pw-footer{position:sticky;bottom:-1px;padding:16px 0 4px;background:Canvas;border-top:1px solid var(--pw-border);margin-top:20px;display:flex;gap:12px;flex-wrap:wrap}.pw-ui .pw-footer button{flex:1}.pw-ui .pw-visually-hidden{position:absolute;width:1px;height:1px;overflow:hidden;clip-path:inset(50%);white-space:nowrap;padding:0;border:0}
@media(prefers-color-scheme:dark){.pw-ui{color-scheme:light dark;--pw-accent:#c4dccb;--pw-on-accent:#1d3528;--pw-border:#718178;--pw-muted:#becdc4;--pw-soft:#27362e}}
@media(max-width:540px){.pw-ui .pw-actions{grid-template-columns:1fr}.pw-ui .pw-header{gap:8px}.pw-ui .pw-footer{padding-bottom:calc(8px + env(safe-area-inset-bottom,0px))}.pw-ui .pw-header .pw-quiet{padding:10px}.pw-ui .pw-header .pw-quiet .pw-action-copy{display:none}.pw-ui .pw-steps{gap:4px;font-size:12px}.pw-ui .pw-steps li{padding:4px 6px}.pw-ui .pw-action:has(small){min-height:88px}}
`;
const iconPaths = {
    camera: ['M14 4h-4L8 7H4a2 2 0 0 0-2 2v10a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2h-4z', 'M16 14a4 4 0 1 1-8 0 4 4 0 0 1 8 0'],
    upload: ['M12 16V3', 'm7 8 5-5 5 5', 'M3 15v4a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-4'],
    document: ['M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z', 'M14 2v6h6', 'M8 13h8', 'M8 17h5'],
    back: ['m12 5-7 7 7 7', 'M5 12h14'], close: ['m6 6 12 12', 'M6 18 18 6'], check: ['m5 12 4 4L19 6'], refresh: ['M20 7v5h-5', 'M4 17v-5h5', 'M6.1 6.1A8 8 0 0 1 20 12', 'M4 12a8 8 0 0 0 13.9 5.9']
};
function uiIcon(name) { const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg'); svg.setAttribute('viewBox', '0 0 24 24'); svg.setAttribute('fill', 'none'); svg.setAttribute('stroke', 'currentColor'); svg.setAttribute('stroke-width', '1.8'); svg.setAttribute('stroke-linecap', 'round'); svg.setAttribute('stroke-linejoin', 'round'); svg.setAttribute('aria-hidden', 'true'); for (const d of iconPaths[name] || iconPaths.document) {
    const path = document.createElementNS(svg.namespaceURI, 'path');
    path.setAttribute('d', d);
    svg.append(path);
} return svg; }
export function stylePanel(root) { root.classList.add('pw-ui'); root.style.fontFamily = 'system-ui,-apple-system,"Segoe UI",sans-serif'; root.style.padding = 'clamp(16px,4vw,32px)'; root.style.maxHeight = '85dvh'; root.style.overflow = 'auto'; if (!root.dataset.paperworkStyled) {
    const style = document.createElement('style');
    style.textContent = panelCSS;
    root.prepend(style);
    root.dataset.paperworkStyled = 'true';
} }
export function actionButton(label, icon = 'document', variant = '', description = '') { const button = document.createElement('button'); button.type = 'button'; button.className = 'pw-action' + (variant ? ' pw-' + variant : ''); button.setAttribute('aria-label', label); button.append(uiIcon(icon)); const copy = document.createElement('span'); copy.className = 'pw-action-copy'; const title = document.createElement('span'); title.textContent = label; copy.append(title); if (description) {
    const detail = document.createElement('small');
    detail.textContent = description;
    copy.append(detail);
} button.append(copy); return button; }
export function panelHeader(root, title, subtitle, step) { stylePanel(root); const heading = document.createElement('header'); heading.className = 'pw-header'; const copy = document.createElement('div'); const brand = document.createElement('p'); brand.className = 'pw-eyebrow'; brand.textContent = 'PAPERWORK'; const h = document.createElement('h2'); h.textContent = title; copy.append(brand, h); if (subtitle) {
    const text = document.createElement('p');
    text.className = 'pw-subtitle';
    text.textContent = subtitle;
    copy.append(text);
} heading.append(copy); root.append(heading); if (step) {
    const steps = document.createElement('ol');
    steps.className = 'pw-steps';
    steps.setAttribute('aria-label', 'Etapas do documento');
    ['Adicionar', 'Revisar', 'Salvar nas notas'].forEach((name, index) => { const item = document.createElement('li'); item.textContent = `${index + 1}. ${name}`; if (index + 1 === step)
        item.setAttribute('aria-current', 'step'); steps.append(item); });
    root.append(steps);
} return heading; }
export function actionGroup() { const group = document.createElement('div'); group.className = 'pw-actions'; return group; }
export function uploadPicker() { const root = document.createElement('div'); const input = document.createElement('input'); input.type = 'file'; input.accept = '.pdf,.png,.jpg,.jpeg,.webp'; input.className = 'pw-visually-hidden'; input.tabIndex = -1; input.setAttribute('aria-label', 'Arquivo para importar'); const button = actionButton('Enviar arquivo', 'upload', '', 'PDF, JPG, PNG ou WebP · até 18 MB'); button.addEventListener('click', () => input.click()); root.append(button, input); return { root, input, button }; }
export function buttonBusy(button, busy) { button.disabled = busy; button.setAttribute('aria-busy', String(busy)); }
let activeCamera;
export function cameraPanel() {
    activeCamera?.close();
    const root = document.createElement('div');
    stylePanel(root);
    root.style.padding = '0';
    const stage = document.createElement('div');
    stage.className = 'pw-camera-stage';
    const placeholder = document.createElement('div');
    placeholder.className = 'pw-camera-placeholder';
    placeholder.append(uiIcon('camera'));
    const hint = document.createElement('div');
    hint.textContent = 'Uma página por foto. Enquadre todo o papel em um local iluminado.';
    placeholder.append(hint);
    const video = document.createElement('video');
    video.autoplay = true;
    video.muted = true;
    video.playsInline = true;
    video.setAttribute('playsinline', '');
    video.hidden = true;
    const image = document.createElement('img');
    image.alt = 'Prévia da página fotografada';
    image.hidden = true;
    stage.append(placeholder, video, image);
    const status = document.createElement('p');
    status.setAttribute('role', 'status');
    status.setAttribute('aria-live', 'polite');
    const start = actionButton('Ativar câmera', 'camera', 'primary');
    const take = actionButton('Fotografar', 'camera', 'primary');
    take.disabled = true;
    take.hidden = true;
    const native = document.createElement('input');
    native.type = 'file';
    native.accept = 'image/jpeg,image/png,image/webp';
    native.setAttribute('capture', 'environment');
    native.className = 'pw-visually-hidden';
    native.tabIndex = -1;
    native.setAttribute('aria-label', 'Câmera nativa ou foto do aparelho');
    const fallback = actionButton('Usar foto do aparelho', 'upload');
    fallback.addEventListener('click', () => native.click());
    const actions = actionGroup();
    actions.append(start, take, fallback);
    const help = document.createElement('p');
    help.className = 'pw-hint';
    help.textContent = 'A foto só será adicionada quando você confirmar. Nenhum envio à IA nesta etapa.';
    let stream = null, selected, closed = false, preview, generation = 0, starting = false, mounted = false;
    let confirmation;
    const change = () => { if (confirmation)
        confirmation.disabled = !selected; };
    const stop = () => { stream?.getTracks().forEach(track => track.stop()); stream = null; video.srcObject = null; video.hidden = true; take.disabled = true; take.hidden = true; start.hidden = false; };
    const clear = () => { selected = undefined; if (preview)
        URL.revokeObjectURL(preview); preview = undefined; image.removeAttribute('src'); image.hidden = true; change(); };
    const show = (file) => { if (closed)
        return; generation++; stop(); clear(); if (file.size > 18_000_000) {
        placeholder.hidden = false;
        status.textContent = 'A foto excede 18 MB. Escolha uma imagem menor.';
        return;
    } if (!['image/jpeg', 'image/png', 'image/webp'].includes(file.type)) {
        placeholder.hidden = false;
        status.textContent = 'Escolha uma foto JPG, PNG ou WebP.';
        return;
    } selected = file; preview = URL.createObjectURL(file); image.src = preview; image.hidden = false; placeholder.hidden = true; start.setAttribute('aria-label', 'Tirar outra foto'); start.querySelector('.pw-action-copy > span').textContent = 'Tirar outra foto'; start.classList.remove('pw-primary'); status.textContent = 'Foto pronta. Confira a imagem e continue para revisar.'; change(); };
    start.addEventListener('click', async () => { if (starting || closed)
        return; starting = true; const ticket = ++generation; stop(); clear(); placeholder.hidden = false; buttonBusy(start, true); status.textContent = 'Aguardando permissão para abrir a câmera…'; try {
        if (!navigator.mediaDevices?.getUserMedia)
            throw new Error('unavailable');
        const incoming = await navigator.mediaDevices.getUserMedia({ video: { facingMode: { ideal: 'environment' } }, audio: false });
        if (closed || ticket !== generation || document.hidden) {
            incoming.getTracks().forEach(t => t.stop());
            if (!closed && ticket === generation)
                status.textContent = 'Câmera pausada. Ative novamente para continuar.';
            return;
        }
        stream = incoming;
        video.srcObject = incoming;
        video.hidden = false;
        await video.play();
        if (closed || ticket !== generation)
            return;
        placeholder.hidden = true;
        start.hidden = true;
        take.hidden = false;
        take.disabled = false;
        take.focus();
        status.textContent = 'Enquadre a página inteira e toque em Fotografar.';
    }
    catch {
        if (!closed && ticket === generation) {
            stop();
            status.textContent = 'Câmera indisponível. Verifique a permissão ou escolha Usar foto do aparelho.';
        }
    }
    finally {
        starting = false;
        buttonBusy(start, false);
    } });
    take.addEventListener('click', () => { if (!video.videoWidth || !video.videoHeight) {
        status.textContent = 'Aguarde a câmera ficar pronta.';
        return;
    } const canvas = document.createElement('canvas'); canvas.width = video.videoWidth; canvas.height = video.videoHeight; const ctx = canvas.getContext('2d'); if (!ctx) {
        status.textContent = 'Não foi possível capturar a imagem.';
        return;
    } const ticket = generation; take.disabled = true; ctx.drawImage(video, 0, 0); canvas.toBlob(blob => { if (closed || ticket !== generation)
        return; if (blob)
        show(new File([blob], 'pagina-' + Date.now() + '.jpg', { type: 'image/jpeg' }));
    else {
        take.disabled = false;
        status.textContent = 'Não foi possível capturar. Tente novamente.';
    } }, 'image/jpeg', .95); });
    native.addEventListener('change', () => { if (native.files?.[0])
        show(native.files[0]); native.value = ''; });
    const onHidden = () => { if (document.hidden && stream) {
        generation++;
        stop();
        placeholder.hidden = false;
        status.textContent = 'Câmera pausada. Ative novamente para continuar.';
    } };
    const onPageHide = () => close();
    document.addEventListener('visibilitychange', onHidden);
    window.addEventListener('pagehide', onPageHide);
    const observer = new MutationObserver(() => { if (root.isConnected)
        mounted = true;
    else if (mounted)
        close(); });
    observer.observe(document.body, { childList: true, subtree: true });
    function close() { if (closed)
        return; closed = true; generation++; stop(); clear(); observer.disconnect(); document.removeEventListener('visibilitychange', onHidden); window.removeEventListener('pagehide', onPageHide); }
    root.append(stage, actions, native, status, help);
    const result = { root, close, bindConfirmation: (button) => { confirmation = button; change(); }, photo: async () => { if (!selected)
            throw new Error('Tire ou selecione uma foto primeiro.'); return { name: selected.name, data: await readDocument(selected) }; } };
    activeCamera = result;
    return result;
}
