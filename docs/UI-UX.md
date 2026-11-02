# Interface: importação e câmera

A Inbox destaca duas ações: **Fotografar página** e **Enviar arquivo**. Ícones complementam rótulos visíveis, sem substituí-los. Arquivos selecionados seguem diretamente à revisão, sem exigir reabrir a Inbox.

A câmera apresenta instruções breves, a área de enquadramento e uma ação principal por estado. Depois da captura, a foto substitui o vídeo e **Usar esta foto** é habilitado. **Tirar outra foto** permite refazer a captura. Permissão pendente, indisponibilidade, erro e importação têm feedback textual. A seleção pelo aparelho continua disponível quando não há câmera no navegador.

Os controles têm alvos de pelo menos 48 × 48 CSS px, espaçamento, foco de teclado visível e estados desabilitados. Em telas estreitas as ações ficam em uma coluna; a confirmação da foto fica ao fim do painel, com área segura para dispositivos móveis. O tema acompanha o esquema de cores do navegador. O estilo é limitado à classe Paperwork, sem alterar o núcleo ou a interface de outras extensões.

A aplicação preserva os bytes dos arquivos selecionados e mantém a captura JPEG na resolução nativa. A câmera é encerrada após a foto, ao remover o painel ou ao sair da página. Resultados tardios de uma solicitação de câmera encerrada são descartados. Importações não são duplicadas por cliques repetidos e não reabrem um painel que o usuário já fechou. Capturar ou importar não envia o documento à IA.

Referências de projeto:

- [WCAG 2.2 — Target Size (Enhanced)](https://www.w3.org/WAI/WCAG22/Understanding/target-size-enhanced.html): referência de 44 CSS px; a interface adota 48 px.
- [WCAG 2.2 — Focus Visible](https://www.w3.org/WAI/WCAG22/Understanding/focus-visible.html): foco perceptível durante operação por teclado.

Essas decisões não constituem uma auditoria completa de conformidade WCAG. As verificações cobrem câmera simulada, bytes do upload, encerramento das trilhas, tamanho dos controles, foco e tema escuro. A Inbox foi conferida no SilverBullet em viewport móvel; os estados de confirmação e as permissões negada/tardia têm regressões DOM sem navegador. A repetição final de captura até revisão no cliente nativo ficou limitada pela inicialização do Chromium, portanto essa rodada não é apresentada como aceite completo em dispositivos reais.
