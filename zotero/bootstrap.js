"use strict";
// Generated JavaScript: edit this TypeScript source and run npm run build.
/* Zotero 7 bootstrapped extension. No remote requests or bibliography rewriting. */
var root, busy = Promise.resolve();
var pref = 'extensions.paperwork.aliasRegistry.v1';
function registry() {
    const value = JSON.parse(Zotero.Prefs.get(pref, true) || '{}');
    if (!value || Array.isArray(value) || typeof value !== 'object')
        throw new Error('Registro de aliases inválido; preserve a preferência antes de recuperar.');
    return Object.assign(Object.create(null), value);
}
function identity(item) {
    const path = Zotero.URI.getItemPath(item);
    const match = /^(users|groups)\/(\d+)\/items\/([A-Z0-9]{8})$/.exec(path);
    if (!match)
        throw new Error('Conecte esta biblioteca à conta Zotero antes de exportar identidades para a API.');
    return { library_type: match[1], library_id: Number(match[2]), item_key: match[3] };
}
function key(item) { return Zotero.URI.getItemPath(item); }
function selected(win) {
    const items = win.ZoteroPane.getSelectedItems().filter(item => item.isRegularItem() && !item.deleted);
    if (!items.length || items.length > 200)
        throw new Error('Selecione entre 1 e 200 itens bibliográficos (não anexos).');
    return items;
}
function copy(text) { Cc['@mozilla.org/widget/clipboardhelper;1'].getService(Ci.nsIClipboardHelper).copyString(text); }
async function create(win) {
    const items = selected(win), reg = registry();
    // Include all current libraries and trash, not just the current selection.
    for (const library of Zotero.Libraries.getAll()) {
        const all = await Zotero.Items.getAll(library.libraryID, true, true);
        for (const item of all) {
            if (!item.isRegularItem())
                continue;
            const alias = PaperworkAliases.read(item.getField('extra'));
            if (alias)
                PaperworkAliases.reserve(reg, alias, key(item));
        }
    }
    const pending = [];
    for (const item of items) {
        if (!Zotero.Libraries.get(item.libraryID).editable)
            throw new Error('Biblioteca somente leitura.');
        identity(item);
        const extra = item.getField('extra') || '';
        const existing = PaperworkAliases.read(extra);
        const alias = existing || PaperworkAliases.allocate(reg, PaperworkAliases.base({ creators: item.getCreators(), date: item.getField('date'), title: item.getField('title') }), key(item));
        pending.push({ item, extra, alias, existing });
    }
    if (pending.some(p => !p.existing) && !Services.prompt.confirm(win, 'Paperwork', 'Acrescentar Paperwork Alias ao campo Extra de ' + pending.filter(p => !p.existing).length + ' item(ns)? Título, autores, citation keys e os outros campos serão preservados.'))
        return;
    // Reserve before writes: a failed save must never free an identifier already seen.
    Zotero.Prefs.set(pref, JSON.stringify(reg), true);
    for (const p of pending) {
        if (!p.existing) {
            p.item.setField('extra', p.extra + (p.extra.endsWith('\n') || !p.extra ? '' : '\n') + 'Paperwork Alias: ' + p.alias);
            await p.item.saveTx();
        }
    }
    copy(pending.map(p => 'Ƶ ' + p.alias).join('\n'));
}
async function exportIndex(win) {
    const reg = registry(), references = [];
    for (const item of selected(win)) {
        const alias = PaperworkAliases.read(item.getField('extra'));
        if (!alias)
            throw new Error('Crie o identificador antes de exportar: ' + item.getField('title'));
        PaperworkAliases.reserve(reg, alias, key(item));
        references.push({ alias, ...identity(item) });
    }
    copy(JSON.stringify({ format: 'paperwork-zotero-aliases-v1', references }, null, 2));
}
function backupRegistry() {
    copy(JSON.stringify({ format: 'paperwork-zotero-registry-v1', reservations: registry() }, null, 2));
}
function restoreRegistry(win) {
    const input = { value: '' };
    if (!Services.prompt.prompt(win, 'Paperwork', 'Cole o backup JSON de reservas. As reservas atuais serão preservadas.', input, null, {}))
        return;
    const value = JSON.parse(input.value), current = registry();
    if (value.format !== 'paperwork-zotero-registry-v1' || !value.reservations || Array.isArray(value.reservations))
        throw new Error('Backup inválido.');
    for (const [alias, identity] of Object.entries(value.reservations)) {
        if (typeof identity !== 'string' || !/^(users|groups)\/\d+\/items\/[A-Z0-9]{8}$/.test(identity))
            throw new Error('Identidade inválida no backup.');
        PaperworkAliases.reserve(current, alias, identity);
    }
    Zotero.Prefs.set(pref, JSON.stringify(current), true);
}
function action(win, operation) {
    busy = busy.then(() => operation(win)).catch(error => { Zotero.logError(error); Services.prompt.alert(win, 'Paperwork', error.message); });
}
function onMainWindowLoad({ window: win }) {
    const menu = win.document.getElementById('zotero-itemmenu');
    if (!menu || win.document.getElementById('paperwork-create-alias'))
        return;
    const entries = [
        ['paperwork-create-alias', 'Paperwork: criar/copiar identificador manuscrito', create],
        ['paperwork-export-index', 'Paperwork: copiar índice selecionado', exportIndex],
        ['paperwork-backup-registry', 'Paperwork: copiar backup de reservas', backupRegistry],
        ['paperwork-restore-registry', 'Paperwork: restaurar reservas', restoreRegistry]
    ];
    for (const [id, label, operation] of entries) {
        const entry = win.document.createXULElement('menuitem');
        entry.id = id;
        entry.setAttribute('label', label);
        entry.addEventListener('command', () => action(win, operation));
        menu.appendChild(entry);
    }
}
function onMainWindowUnload({ window: win }) {
    for (const id of ['paperwork-create-alias', 'paperwork-export-index', 'paperwork-backup-registry', 'paperwork-restore-registry'])
        win.document.getElementById(id)?.remove();
}
async function startup({ rootURI }) {
    root = rootURI;
    Services.scriptloader.loadSubScript(rootURI + 'aliases.js', globalThis);
    for (const win of Zotero.getMainWindows())
        onMainWindowLoad({ window: win });
}
async function shutdown() {
    await busy;
    for (const win of Zotero.getMainWindows())
        onMainWindowUnload({ window: win });
}
function install() { }
function uninstall() { } // Keep Extra and registry: handwritten citations must survive uninstall.
