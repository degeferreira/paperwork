/** Minimal Zotero 7 / Mozilla host contracts used by this extension. */
interface ZoteroItem {
  libraryID: number;
  deleted: boolean;
  isRegularItem(): boolean;
  getField(name: string): string;
  getCreators(): AliasCreator[];
  setField(name: string, value: string): void;
  saveTx(): Promise<unknown>;
}
interface ZoteroMenuEntry {
  id: string;
  setAttribute(name: string, value: string): void;
  addEventListener(name: string, listener: () => void): void;
  appendChild(child: ZoteroMenuEntry): void;
  remove(): void;
}
interface ZoteroWindow {
  ZoteroPane: {getSelectedItems(): ZoteroItem[]};
  document: {
    getElementById(id: string): ZoteroMenuEntry | null;
    createXULElement(name: string): ZoteroMenuEntry;
  };
}
declare const Zotero: {
  Prefs: {get(name: string, global: boolean): string | undefined; set(name: string, value: string, global: boolean): void};
  URI: {getItemPath(item: ZoteroItem): string};
  Libraries: {getAll(): {libraryID: number}[]; get(id: number): {editable: boolean}};
  Items: {getAll(id: number, onlyTopLevel: boolean, includeDeleted: boolean): Promise<ZoteroItem[]>};
  getMainWindows(): ZoteroWindow[];
  logError(error: unknown): void;
};
declare const Services: {
  prompt: {
    confirm(win: ZoteroWindow, title: string, message: string): boolean;
    prompt(win: ZoteroWindow, title: string, message: string, input: {value: string}, checkLabel: null, check: object): boolean;
    alert(win: ZoteroWindow, title: string, message: string): void;
  };
  scriptloader: {loadSubScript(uri: string, scope: typeof globalThis): void};
};
declare const Cc: Record<string, {getService(service: unknown): {copyString(text: string): void}}>;
declare const Ci: {nsIClipboardHelper: unknown};
