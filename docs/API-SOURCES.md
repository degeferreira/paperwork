# APIs e fontes do Paperwork

## Fronteiras do projeto

A Library usa Space Lua nas páginas Markdown e módulos JavaScript gerados de
TypeScript estrito em `src/Library/Scripts`. O serviço Python existente continua
responsável por armazenamento, jobs, PFM e incorporação. O complemento Zotero é
independente do serviço e não envia o índice automaticamente.

## Código de referência neste repositório

| Fluxo | Fonte |
|---|---|
| Câmera, arquivo, impressão, revisão | `src/Library/Scripts/Documents.ts` |
| Comandos e telas SilverBullet | `Library/Paperwork.md` |
| Aliases e colisões | `src/zotero/aliases.ts` |
| Zotero: menus, Extra, reservas e clipboard | `src/zotero/bootstrap.ts` |
| Contratos mínimos do host Zotero 7 | `src/zotero/host.d.ts` |
| XPI determinístico | `scripts/build-zotero.cjs` |
| APIs de importação e revisão | `paperwork/api.py` |
| Provedores de transcrição | `paperwork/transcription.py` |

Os contratos TypeScript descrevem apenas as APIs utilizadas. Não substituem uma
validação no Zotero desktop real. Os testes `tests/zotero-addon.cjs` simulam o
host; `tests/browser-camera.cjs` usa Chromium com câmera sintética. Nenhum desses
testes demonstra qualidade de OCR de manuscritos nem compatibilidade de câmera física.

## Documentação externa

- [Desenvolvimento de extensões Zotero 7](https://www.zotero.org/support/dev/zotero_7_for_developers)
- [Código Zotero: itens](https://github.com/zotero/zotero/blob/main/chrome/content/zotero/xpcom/data/items.js)
- [Código Zotero: identidade URI](https://github.com/zotero/zotero/blob/main/chrome/content/zotero/xpcom/uri.js)

Para configurar usuários, permissões, importação e backup de reservas, consulte
[o guia do complemento](../zotero/README.md). Para o fluxo de revisão no Space,
consulte [SILVERBULLET-FLOWS.md](SILVERBULLET-FLOWS.md).
