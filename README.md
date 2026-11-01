<p align="center"><img src="assets/logo.svg" width="112" alt="Logo Paperwork: folha, traço de caneta e ponto de memória"></p>
<h1 align="center">Paperwork</h1>
<p align="center">O papel para pensar; o digital para lembrar.</p>

Paperwork é uma extensão **analog-first para SilverBullet 2.11.1**. Você escreve em papel comum, fotografa ou digitaliza, escolhe a IA de transcrição, revisa e acrescenta o conteúdo às páginas que já usa. Funciona também com transcrição manual, sem IA.

**Seu diário é um só:** tanto a escrita digital quanto o manuscrito revisado usam `Daily/AAAA-MM-DD`. Não existe um segundo diário Paperwork. A incorporação acrescenta blocos com identificadores e origem, sem substituir seu texto. Na instalação com Relay, o botão do diário chama a função já existente; não acrescentamos outro comando de diário.

Esta versão é um **MVP experimental**. Há testes automatizados, inclusive captura em Chromium com câmera simulada e APIs simuladas. Não há garantia de reconhecimento perfeito de caligrafia nem validação gráfica em todas as plataformas.

## Começar a usar

Recarregue o SilverBullet após instalar/atualizar a biblioteca. Na paleta de comandos:

1. Abra **Paperwork: Configurações** e selecione a IA (ou transcrição manual).
2. Abra **Paperwork: Inbox**. Escolha um PDF multipágina, PNG, JPG ou WebP e clique em **Importar original**, ou clique em **Abrir câmera / fotografar página**.
3. Abra o documento na Inbox. O original e a área de transcrição aparecem lado a lado.
4. Clique em **Transcrever com provedor configurado**. Para serviço externo, autorize esse documento. Acompanhe em **Consultar processamento** e depois **Carregar proposta existente**.
5. Confira o texto, a data, os filhos e as pendências. Use **Interpretar e revisar** depois de corrigir o texto.
6. Marque **Revisei a transcrição e a estrutura propostas** e clique em **Incorporar nos diários e notas**.

A proposta não muda suas notas antes da incorporação. Repetir a mesma incorporação concluída não duplica blocos nem recria uma página que você moveu. Alterar a árvore de um documento já incorporado exige reconciliação manual nesta versão.

### Câmera dentro do SilverBullet

Na Inbox, **Abrir câmera / fotografar página → Ligar câmera → Fotografar → Importar foto**. Autorize a câmera no navegador. Há uma prévia antes de importar; fotografar não envia o documento à IA. A câmera é encerrada depois da captura ou ao fechar o painel.

O acesso direto depende de HTTPS ou localhost e de suporte/permissão do navegador. O seletor alternativo solicita a câmera traseira nativa no celular; se o navegador não oferecer câmera, ele permite escolher uma foto já salva. Permissões do aplicativo/PWA também podem precisar ser liberadas no sistema.

Cada foto importada é um original independente. Para preservar contexto de diário entre várias folhas em um só processamento, use **um PDF multipágina**. Esta versão não reúne automaticamente fotos em PDF e não faz recorte, correção de perspectiva ou scanner multipágina.

Limite: **18 MB por arquivo**. Originais enviados por arquivo não são recomprimidos. A câmera codifica o quadro capturado em JPEG; esses bytes são preservados após a importação.

### Escolher a IA

O seletor oferece:

| Opção | Credencial | Configuração |
|---|---|---|
| Sem IA | Nenhuma | Cole ou escreva a transcrição |
| Google Gemini | `gemini_key` | Identificador de modelo capaz de ler imagem/PDF |
| Anthropic Claude | `claude_key` | Identificador de modelo com a capacidade necessária |
| ChatGPT / OpenAI (API) | `openai_key` | Identificador de modelo multimodal compatível com Responses |
| Outro compatível com OpenAI | `compatible_key`, quando exigida | Endpoint `/v1` e modelo com visão/JSON |
| Modelo local / Ollama compatível | `local_key`, quando exigida | Endpoint `/v1` e modelo com visão/JSON |

As opções usam **APIs**, não a sessão do navegador de ChatGPT/Claude/Gemini. Informe sua chave de API no campo correspondente e salve-a; depois selecione o provedor, informe o modelo e salve as configurações. A escolha não faz chamadas nem gera uma cobrança de teste. Os modelos não são fixados automaticamente, pois disponibilidade e capacidades dependem do provedor/conta.

Os campos de modelo e endpoint lembram os perfis salvos ao alternar. Chaves ficam em arquivos privados do servidor, fora do Markdown e sem serem retornadas ao navegador. Compartilhamento com Relay reutiliza as conexões SilverBullet/Vikunja/Zotero; não substitui suas credenciais.

Documentos começam como somente locais. Para APIs externas, habilite **Permitir provedores externos** nas configurações e confirme o envio na revisão do documento. Um endpoint em outro host também exige essa autorização conservadora, mesmo em rede privada. Sem IA configurada, importação, transcrição manual, busca e consulta continuam funcionando.

## PFM: o markup para escrever à mão

Você não precisa desenhar formulários nem usar papel especial. Os marcadores ajudam o parser; em papel livre, atribua a data ou o título na revisão. A IA transcreve, mas não deve inventar datas, relações ou identificadores.

| Escreva | Significado |
|---|---|
| `// 2026-11-01` | Começa um segmento no diário dessa data |
| `//end` | Encerra o segmento; uma nova data também encerra o anterior |
| `• texto` ou `- texto` | Bullet comum |
| Recuo | Bullet filho; o mesmo recuo indica irmãos |
| `> • texto` | Filho explícito, mesmo escrito junto à margem |
| `>> ? texto` | Neto explícito |
| `□ fazer algo` | Tarefa aberta |
| `× T00001` | Atualização de conclusão de tarefa local identificada |
| `! ideia` | Ideia importante |
| `? pergunta` | Pergunta |
| `@tese` | Projeto, ligado a `Projects/tese` |
| `@@ana` | Pessoa, ligada a `People/ana` |
| `#memoria` | Tag/conceito |
| `Ƶ Silva24-Mem p.12` | Obra Zotero por alias; página opcional |
| `:: Título` e `::end` | Começa/encerra uma nota independente |
| `\> literal` | Escapa um marcador estrutural no início da linha |
| `//continue ID` | Continuidade explícita; ainda exige resolução manual |

Use nomes curtos sem espaços para projetos e pessoas: `@projeto-tese`, `@@ana-silva`. `@tese` continua significando projeto; `@@ana` não é interpretado como projeto. Pessoa e projeto geram links nativos separados, aproveitando os backlinks do SilverBullet. Tags, projetos, pessoas e referências são herdados pelos filhos quando não há substituição explícita. E-mail como `ana@example.org` não é uma menção de projeto. Os marcadores de menção precisam estar no início do texto ou após espaço; outros usos de `@` permanecem texto.

```text
// 2026-11-01
• Leitura Ƶ Silva24-Mem p.12 @tese #memoria
   • Argumento principal do autor
> ! Minha interpretação
>> ? Qual é a evidência?
• Conversa @@ana @tese
> □ Preparar roteiro da próxima entrevista
//end

:: Ideias para um ensaio
Texto livre de uma nota independente.
::end
```

O prefixo explícito `>` define profundidade: um `>` é filho, dois são neto. Recuo e prefixo conflitantes, datas inválidas e filhos sem pai entram em revisão. Texto sem novo bullet continua o anterior; não se deve reparentear silenciosamente. Um segmento pode atravessar páginas do **mesmo PDF**. Datas diferentes vão para suas respectivas Dailies, não para a data em que o OCR foi executado. Para transcrição manual de várias páginas, separe-as com uma linha `---PAGE---`.

## Zotero: identificador curto para escrever no caderno

A [extensão desktop complementar](zotero/README.md), neste mesmo repositório, cria um alias como **`Ƶ Silva24-Mem`**. Ela preserva a identidade Zotero real, acrescenta `Paperwork Alias: Silva24-Mem` ao Extra após confirmação e oferece cópia do índice para o Paperwork.

Baixe o [XPI para Zotero 7](zotero/dist/paperwork-zotero-0.2.0.xpi?raw=true), instale pelo gerenciador de extensões do Zotero e siga o [passo a passo](zotero/README.md). No SilverBullet, **Paperwork: Importar identificadores do Zotero** recebe o índice JSON. **Paperwork: Abrir Literature Note** importa/abre a nota do alias e associa o link nativo. Na instalação com Relay, a mesma biblioteca pessoal reutiliza o importador e as notas existentes.

O alias é persistente, não uma citation key do Better BibTeX. Sufixos numéricos resolvem colisões; um alias existente nunca é reassociado automaticamente. Zotero continua sendo a autoridade bibliográfica. Para outra máquina, transfira também o backup de reservas do complemento, descrito no README dele.

## Tarefas e retorno ao papel

As tarefas manuscritas recebem IDs locais próprios e podem ser usadas sem Vikunja. Depois de revisar, use **Exportar tarefas revisadas ao Vikunja** e informe o projeto remoto. Apenas filhos que são tarefas viram subtarefas; comentários e perguntas não viram tarefas.

Os símbolos `□` são preservados em vez de criar checkboxes nativos que disparariam uma segunda exportação pelo Relay. **Paperwork: Tarefas locais** permite consultar, concluir e imprimir uma folha com IDs. Depois, uma captura com `× T00001` permite revisar a conclusão sem alterar o scan original. A versão atual tem exportação/reconciliação manual; não tem sincronização bidirecional automática.

## Onde ficam os dados

O serviço guarda originais imutáveis por SHA-256 em `$PAPERWORK_HOME/originals/`; o nome original e os metadados ficam em `$PAPERWORK_HOME/paperwork.sqlite`. As páginas incorporadas são Markdown comum **no seu Space**. O SQLite guarda o índice e o histórico operacional; não substitui o diário Markdown por outro diário oculto.

No pacote Docker independente, `$PAPERWORK_HOME` é `/paperwork`, montado de `runtime/paperwork`. No pacote combinado, conexões vêm do Relay, mas o serviço Paperwork permanece separado. As notas documentais ficam em `Paperwork/Documents/`; notas independentes em `Paperwork/Pages/`. Na revisão, **Abrir original preservado** abre o arquivo autenticado.

**Paperwork: Criar backup** captura banco, originais e notas com manifesto e hashes. A CLI permite verificar/restaurar em destinos novos: `python -m paperwork.cli --help`. Backup não é sincronização concorrente: não coloque o SQLite ativo em uma pasta compartilhada entre máquinas.

## Instalação em outra máquina

A extensão tem duas partes: biblioteca **Space Lua/JavaScript no SilverBullet** e serviço auxiliar **Python**, para armazenamento, PDFs e jobs. Python é uma escolha deste projeto, não uma exigência do SilverBullet. Instalar apenas o Manifest da biblioteca não instala o serviço.

Para uma instalação Docker **nova**:

```sh
git clone https://github.com/degeferreira/paperwork.git
cd paperwork
python3 package.py /caminho/instalacao
python3 bootstrap.py /caminho/instalacao
cd /caminho/instalacao
docker compose up -d --build
```

Configure a autenticação do SilverBullet e gere um token de acesso ao Space; guarde-o pelas configurações Paperwork. Não execute bootstrap sobre uma instalação existente: ele recusa reinicialização de runtime. O endereço do endpoint é visto do contêiner, portanto `127.0.0.1` não é outro computador nem outro contêiner.

Para combinar com Relay, mantenha os checkouts irmãos `silverbullet-relay` e `paperwork`, e use o `package.py DESTINO --paperwork` do Relay. O pacote escolhe porta interna **8089** para Paperwork, mantendo **8088** para Relay. Não ative também `HUB_PAPERWORK`. Em uma instalação existente, faça backup e migre as montagens/bibliotecas; não substitua dados ou credenciais por exemplos.

## Validação e limites

Testes cobrem PFM, herança, citações tipadas, originais, deduplicação, escrita condicional, tarefas, políticas de envio, APIs simuladas, backup e índice Zotero. O teste de câmera usa Chromium e dispositivo sintético. O XPI tem testes de contrato dos hooks Zotero e da atribuição de aliases; **ainda não foi exercitado no Zotero desktop real**. IA não foi testada com manuscritos pessoais. Suporte a câmera física no seu aparelho depende da validação de permissões e compatibilidade.

Permanecem fora desta versão: scanner multipágina interno, regiões gráficas, QR, miniaturas avançadas, resolução automática de `//continue`, atualização reconciliada de árvores já incorporadas e todas as plataformas do aplicativo original.

```sh
# Dependências Python: requirements.txt, em ambiente virtual
PYTHONPATH=. python -m unittest discover -s tests -q
npm run test:zotero
npm run build:zotero
# Com playwright-core e Chromium instalados:
NODE_PATH=/caminho/node_modules node tests/browser-camera.cjs
```

[Fluxos adaptados ao SilverBullet](docs/SILVERBULLET-FLOWS.md) · [Arquitetura das APIs e fontes](docs/API-SOURCES.md) · [Repositório](https://github.com/degeferreira/paperwork)

## Desenvolvimento TypeScript

Os módulos de navegador têm fonte em `src/Library/Scripts/*.ts`, com verificação
`strict`. O build mantém os caminhos `Library/Scripts/*.js` usados pelo Space Lua
 e pelo empacotador. Esses arquivos JavaScript gerados permanecem no repositório
para instalação sem Node.js; edite a fonte TypeScript e reconstrua antes de empacotar.
Space Lua permanece nas páginas da Library.

```sh
npm ci
npm run check
npm run build
```

O compilador é fixado no `package-lock.json`. O build não instala bibliotecas nem
altera serviços em execução. Os testes existentes consomem o JavaScript gerado.

O complemento Zotero tem fonte em `src/zotero/*.ts`; `host.d.ts` descreve a parte
da API Zotero/Mozilla utilizada. `npm run build:zotero` compila primeiro e cria o
XPI e seu SHA-256, com ordem e timestamps ZIP fixos. Repetir o comando com as
mesmas fontes e o mesmo compilador gera os mesmos bytes. Não edite diretamente
`zotero/aliases.js` ou `zotero/bootstrap.js`.

O logo vetorial em [`assets/logo.svg`](assets/logo.svg) integra este projeto e
segue a licença do repositório. Representa uma folha, escrita e um ponto de memória.

### Go migration development

The Go backend is under validation and has not replaced the Python release.
Build the standalone API with `go build -o paperwork-go ./cmd/paperwork`.
For an isolated smoke test, use a new `PAPERWORK_HOME` and
`PAPERWORK_LISTEN=127.0.0.1:18089`. The default listener is `127.0.0.1:8089`.
Do not run it against the active library while another worker is using that state.
The Go tests include fixtures from the existing Python PFM parser and temporary
SQLite libraries; they do not read production documents or credentials.
Daily/Zotero ownership transfer and release packaging are still pending.

### Estado da implementação Go/TypeScript

Fontes Go e TypeScript em desenvolvimento; a presença destes fontes não constitui uma release pronta para substituir o serviço Python. A transferência exclusiva de Daily/Zotero, os consumidores do estado migrado e a instalação limpa ainda precisam ser concluídos antes do corte de produção. Os bancos e credenciais de produção não fazem parte deste repositório.
