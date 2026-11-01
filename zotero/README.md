# Paperwork para Zotero 7

Extensão desktop complementar ao Paperwork, no mesmo repositório. Ela cria **um alias manuscrito legível**, não uma nova identidade bibliográfica e não uma citation key do Better BibTeX.

## O que ela cria

Ao selecionar uma obra de Silva, publicada em 2024, com título começando por Memória, o identificador proposto é:

```text
Ƶ Silva24-Mem
```

`Ƶ` é o marcador escrito no caderno; `Silva24-Mem` é o alias. No campo **Extra** do Zotero fica uma linha:

```text
Paperwork Alias: Silva24-Mem
```

A geração usa sobrenome, ano e três letras da primeira palavra significativa do título. Acentos são simplificados. Uma colisão recebe sufixo: `Silva24-Mem-2`. Uma vez atribuído, o alias não muda quando você altera título ou autor. Aliases existentes são preservados; ambiguidades bloqueiam a operação.

O índice exportado associa o alias à identidade verdadeira `{library_type, library_id, item_key}`. É essa identidade que resolve a obra — não similaridade de título nem adivinhação da IA. A biblioteca precisa ter identidade de conta/grupo Zotero para usar a API. O plugin não envia dados ao Paperwork automaticamente e não exige chave de API.

## Instalar

Baixe [paperwork-zotero-0.2.0.xpi](dist/paperwork-zotero-0.2.0.xpi?raw=true). No Zotero, abra **Ferramentas → Extensões**, use a engrenagem e **Instalar extensão de arquivo**. Escolha o XPI. O manifesto atualmente aceita **Zotero 7.0.x**. Não foi validado no Zotero 8.

O instalador foi construído e o ciclo de vida/API tem testes de contrato. A instalação gráfica no seu Zotero ainda precisa ser validada; não há Zotero desktop instalado neste Raspberry.

## Usar com SilverBullet

1. Selecione até 200 obras (itens bibliográficos, não anexos).
2. No menu de contexto, escolha **Paperwork: criar/copiar identificador manuscrito**. A extensão pede confirmação antes de acrescentar linhas ao Extra e copia os identificadores.
3. Escolha **Paperwork: copiar índice selecionado**.
4. No SilverBullet, execute **Paperwork: Importar identificadores do Zotero**, cole o JSON e importe.
5. Use **Paperwork: Abrir Literature Note** para importar ou abrir a nota bibliográfica do alias. A instalação com Relay reutiliza a nota da mesma biblioteca pessoal.
6. Escreva no papel: `• Argumento Ƶ Silva24-Mem p.12`. Depois da revisão, o bloco no diário aponta para a Literature Note nativa.

## Preservação e outra máquina

Apenas o campo Extra é acrescido, após confirmação. Título, autores, DOI, PDFs, outros campos e citation keys são preservados. Extra acompanha a sincronização normal dos itens no Zotero.

O registro de reservas fica na preferência local `extensions.paperwork.aliasRegistry.v1`. Ele é mantido ao desinstalar: retirar o alias de um item não libera a reserva. Para outra máquina, use **copiar backup de reservas**, guarde o JSON e use **restaurar reservas** no novo perfil. O restore mescla reservas e recusa conflitos antes de gravar. Esse backup é separado do índice selecionado que você cola no SilverBullet.

A unicidade é verificada nas bibliotecas locais e no índice Paperwork; não é um serviço global de atribuição. Evite atribuir aliases simultaneamente em máquinas não sincronizadas. Se surgir conflito após sync, corrija-o explicitamente; o Paperwork não reassocia citações automaticamente. As reservas não são apagadas quando uma obra vai para a lixeira.

## Construção e testes

```sh
npm ci
npm run check
npm run build:zotero
npm run test:zotero
```

O XPI contém somente `manifest.json`, `bootstrap.js` e `aliases.js`. Não contém bancos, credenciais, serviço Python nem originais.

A implementação usa os hooks documentados de [plugins Zotero 7](https://www.zotero.org/support/dev/zotero_7_for_developers), e as APIs de [itens](https://github.com/zotero/zotero/blob/main/chrome/content/zotero/xpcom/data/items.js) e [identidades URI](https://github.com/zotero/zotero/blob/main/chrome/content/zotero/xpcom/uri.js).

As fontes editáveis estão em `../src/zotero/`: regras puras em `aliases.ts`,
ciclo de vida e menus em `bootstrap.ts`, contratos do host em `host.d.ts`.
O build mantém `aliases.js` e `bootstrap.js` neste diretório, carregados como
scripts clássicos pelo Zotero (sem módulos ES ou dependências de runtime).
O construtor `../scripts/build-zotero.cjs` usa entradas em ordem fixa e data ZIP
1980-01-01 para gerar o mesmo XPI byte a byte a partir das mesmas fontes.
O arquivo `.sha256` acompanha o artefato; confira com `sha256sum -c` dentro de `dist`.
O antigo `build.py` permanece disponível por compatibilidade, mas não é a rota
reprodutível: execute `npm run build:zotero` para compilar as fontes antes de empacotar.
