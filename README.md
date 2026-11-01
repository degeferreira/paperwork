# Paperwork

MVP experimental analog-first para SilverBullet 2.11.1. O papel é a origem; a transcrição revisada entra nas páginas nativas, mantendo um link para o original imutável. Não implementa toda a especificação do produto.

## Fluxo

1. Abra **Paperwork: Inbox** e importe PDF ou imagem (até 18 MB por arquivo).
2. Preserve o original; informe uma transcrição manual ou escolha IA nas configurações. Documentos começam restritos a processamento local; envio externo exige autorização explícita.
3. Revise transcrição, datas, hierarquia e possíveis duplicatas. `// AAAA-MM-DD`, `//end`, recuo, `>` e `>>` estruturam o Journal; `:: Título` abre página independente.
4. Incorpore a revisão exata nas páginas Markdown, sem substituir texto pessoal. Repetir uma incorporação concluída não recria páginas movidas. Revisões posteriores de documentos já incorporados exigem reconciliação manual.
5. Registre aliases `Ƶ` para itens Zotero e abra Literature Notes. Tarefas têm identidade local; a exportação manual para Vikunja usa mapeamento persistente e prevenção de duplicatas.
6. Consulte tarefas locais, imprima uma folha com IDs e importe a captura posterior para revisar marcações `× ID`. Use busca e links para abrir os originais.

Há histórico de processamento, fila persistente, visualização lado a lado, pesquisa FTS5 e backup verificável de banco, originais e notas. Adaptadores disponíveis: Gemini e endpoint local compatível; PDF local usa Poppler. Transcrições corrigidas são preservadas.

Veja os [fluxos adaptados ao SilverBullet](docs/SILVERBULLET-FLOWS.md).

## Instalação limpa independente

No checkout:

```sh
python3 package.py /caminho/instalacao
python3 bootstrap.py /caminho/instalacao
cd /caminho/instalacao
docker compose up -d --build
```

Configure a autenticação do SilverBullet e o token de acesso ao Space em arquivo privado. Consulte o `config.example.json` e abra **Paperwork: Configurações**. Não execute bootstrap sobre instalação existente. A interface `Library/Manifest.md` pode ser instalada pelo gerenciador nativo, mas o serviço auxiliar também é necessário.

Sem Relay, o serviço usa porta 8088. Com Relay, o pacote combinado usa **outro serviço, na porta 8089**, reutilizando URLs e credenciais ativas por montagens locais somente de leitura. Não ative `HUB_PAPERWORK` junto deste modo. Credenciais próprias de IA ficam fora do Markdown.

O processo precisa compartilhar o loopback alcançado pelo proxy do SilverBullet. Nenhuma porta adicional precisa ser publicada. Operação local sem IA mantém edição, importação manual, pesquisa e consulta; funcionamento offline depende também do acesso local e do cache do próprio SilverBullet.

## Limites e validação

PDF sintético, instalação limpa real, revisão/incorporação e backup passaram por testes. APIs de IA/Vikunja/Zotero têm testes simulados; qualidade de caligrafia e uso visual em navegador/mobile ainda exigem validação real. Não houve envio de manuscritos pessoais.

Permanecem futuros: seleção gráfica de regiões, miniaturas avançadas, QR, resolução completa de `//continue`, sincronização bidirecional automática, edição reconciliada de árvores já incorporadas e todas as plataformas do aplicativo original. O MVP usa o editor, páginas, links e proxy nativos do SilverBullet; não exige Flutter, SCI nem outro aplicativo desktop.

Backup e restauração pela CLI estão em `python -m paperwork.cli --help`; restauração exige destinos novos e verifica hashes. Não sincronize um SQLite ativo entre máquinas por pasta compartilhada.

Repositório: https://github.com/degeferreira/paperwork

## Testes

Instale as dependências do backend em ambiente virtual e execute:

```sh
PYTHONPATH=. python -m unittest discover -s tests -q
```
