---
tags: meta
---

```space-lua
-- priority: 3
paperwork = {}
function paperwork.home()
  local root = dom.div {}
  for _, item in ipairs({{"Diário de hoje", function() paperwork.daily() end}, {"Inbox de digitalizações", function() paperwork.inbox() end},
    {"Tarefas locais", function() paperwork.tasks() end}, {"Buscar documentos", function() paperwork.search() end}, {"Configurações", function() paperwork.settings() end}}) do
    local action = item[2]
    root.appendChild(dom.button {__rawText = item[1], onclick = action})
  end
  js.importFromSpace("Library/Paperwork/Scripts/Documents.js").stylePanel(root)
  return widget.htmlBlock(root)
end
function paperwork.call(path, method, body)
  local port = tonumber(config.get("paperwork.port")) or 8089
  local options = {method = method or "GET"}
  if body then options.body = body; options.headers = {["Content-Type"] = "application/json"} end
  local address = "http://127.0.0.1:" .. tostring(port) .. "/v1/paperwork/" .. path
  local connection = clientStore.get("personal-hub.desktop.v1")
  if connection then
    address = (connection.server or config.get("relay.legacyServer") or "") .. "/.proxy/127.0.0.1:" .. tostring(port) .. "/v1/paperwork/" .. path
    options.headers = {Authorization = "Bearer " .. connection.token}
    if body then options.headers["X-Proxy-Header-Content-Type"] = "application/json" end
  end
  local response = net.proxyFetch(address, options)
  if connection and response.status == 200 then response.status = tonumber((response.headers or {})["x-proxy-status-code"] or 200) end
  if response.status < 200 or response.status >= 300 then
    local message = type(response.body) == "table" and response.body.detail or nil
    error(type(message) == "string" and message or "Paperwork: HTTP " .. tostring(response.status))
  end
  return response.body
end

function paperwork.inbox()
  local root = dom.div {}
  local helpers = js.importFromSpace("Library/Paperwork/Scripts/Documents.js")
  local header = helpers.panelHeader(root, "Adicionar um documento", "Fotografe uma página ou envie um arquivo para começar.", 1)
  local close = helpers.actionButton("Fechar", "close", "quiet")
  close.addEventListener("click", function() editor.hidePanel("modal") end)
  header.appendChild(close)
  local status = dom.p {role = "status", ["aria-live"] = "polite"}
  local actions = helpers.actionGroup()
  local camera = helpers.actionButton("Fotografar página", "camera", "primary", "Use a câmera do celular ou computador")
  camera.addEventListener("click", paperwork.camera)
  actions.appendChild(camera)
  local picker = helpers.uploadPicker()
  actions.appendChild(picker.root); root.appendChild(actions)
  local busy = false
  picker.input.addEventListener("change", function()
    local selected = picker.input.files[0]
    if not selected or busy then return end
    if selected.size > 18000000 then status.textContent = "O arquivo excede 18 MB. Escolha um arquivo menor."; picker.input.value = ""; return end
    busy = true; helpers.buttonBusy(picker.button, true); helpers.buttonBusy(camera, true)
    status.textContent = "Importando " .. selected.name .. "…"
    local loaded, encoded = pcall(helpers.readDocument, selected)
    if not root.isConnected then return end
    if not loaded then
      status.textContent = tostring(encoded)
    else
      local ok, result = pcall(paperwork.call, "documents", "POST", {name = selected.name, data = encoded})
      if not root.isConnected then return end
      if ok then
        busy = false; paperwork.review({id = result.id, name = selected.name}); return
      end
      status.textContent = tostring(result)
    end
    busy = false; helpers.buttonBusy(picker.button, false); helpers.buttonBusy(camera, false); picker.input.value = ""
  end)
  root.appendChild(dom.p {class = "pw-hint", __rawText = "O original fica preservado. Você revisa o conteúdo antes de salvar nas notas."})
  root.appendChild(status)
  root.appendChild(dom.h3 {__rawText = "Documentos para revisar"})
  local ok, docs = pcall(paperwork.call, "documents")
  if ok then
    if #docs == 0 then root.appendChild(dom.div {class = "pw-empty", __rawText = "Sua caixa de entrada está vazia. Adicione seu primeiro documento acima."}) end
    for _, row in ipairs(docs) do
      local document = row
      local button = helpers.actionButton(document.name, "document", "", "Abrir documento e revisão")
      button.classList.add("pw-document")
      button.addEventListener("click", function() paperwork.review(document) end)
      root.appendChild(button)
    end
  else status.textContent = "Não foi possível carregar os documentos. " .. tostring(docs) end
  js.importFromSpace("Library/Paperwork/Scripts/Documents.js").stylePanel(root)
  editor.showPanel("modal", "5vh 3vw", root)
end
function paperwork.camera()
  local root = dom.div {}
  local helpers = js.importFromSpace("Library/Paperwork/Scripts/Documents.js")
  local header = helpers.panelHeader(root, "Fotografar uma página", "Ative a câmera, enquadre o papel e confira a foto antes de continuar.", 1)
  local capture = helpers.cameraPanel()
  root.appendChild(capture.root)
  local status = dom.p {role = "status", ["aria-live"] = "polite"}
  local footer = dom.div {class = "pw-footer"}
  local back = helpers.actionButton("Voltar", "back", "quiet")
  local next = helpers.actionButton("Usar esta foto", "check", "primary")
  next.disabled = true
  local busy = false
  capture.bindConfirmation(next)
  local function leave() if busy then return end; capture.close(); paperwork.inbox() end
  back.addEventListener("click", leave)
  local close = helpers.actionButton("Fechar", "close", "quiet")
  close.addEventListener("click", function() if busy then return end; capture.close(); editor.hidePanel("modal") end)
  header.appendChild(close)
  root.addEventListener("keydown", function(event) if event.key == "Escape" then event.preventDefault(); event.stopPropagation(); leave() end end)
  next.addEventListener("click", function()
    if busy then return end
    busy = true; helpers.buttonBusy(next, true); helpers.buttonBusy(back, true); helpers.buttonBusy(close, true)
    status.textContent = "Preservando sua foto…"
    local ok, photo = pcall(capture.photo)
    if not root.isConnected then return end
    if ok then
      local imported, result = pcall(paperwork.call, "documents", "POST", {name = photo.name, data = photo.data})
      if not root.isConnected then return end
      if imported then capture.close(); paperwork.review({id = result.id, name = photo.name}); return end
      status.textContent = tostring(result)
    else status.textContent = tostring(photo) end
    busy = false; helpers.buttonBusy(next, false); helpers.buttonBusy(back, false); helpers.buttonBusy(close, false)
  end)
  root.appendChild(status); footer.appendChild(back); footer.appendChild(next); root.appendChild(footer)
  js.importFromSpace("Library/Paperwork/Scripts/Documents.js").stylePanel(root)
  editor.showPanel("modal", "5vh 3vw", root)
end

function paperwork.review(document, startPage)
  local root = dom.div {style = "padding:1em;background:Canvas;color:CanvasText;max-height:85vh;overflow:auto"}
  local ui = js.importFromSpace("Library/Paperwork/Scripts/Documents.js")
  ui.panelHeader(root, "Revisar documento", document.name, 2)
  local sourceLink = dom.a {__rawText = "Abrir original preservado", href = "/.proxy/127.0.0.1:8089/v1/paperwork/documents/" .. document.id .. "/original", target = "_blank", rel = "noopener"}
  root.appendChild(sourceLink)
  root.appendChild(dom.p {__rawText = "Cole a transcrição PFM. Separe páginas com uma linha contendo ---PAGE---. A proposta ainda não altera suas notas."})
  local text = dom.textarea {style = "width:100%;height:35vh"}
  local draftKey = "paperwork.draft.v1:" .. document.id
  local savedDraft = clientStore.get(draftKey)
  if savedDraft then text.value = savedDraft end
  root.appendChild(dom.p {__rawText = "Papel sem marcações: atribua uma data ou um título abaixo. Isso organiza o conteúdo sem acrescentar marcadores à transcrição fiel."})
  local assignedDate = dom.input {type = "date", ["aria-label"] = "Data de escrita atribuída"}
  local assignedTitle = dom.input {type = "text", placeholder = "Ou título de nota independente"}
  root.appendChild(assignedDate); root.appendChild(assignedTitle)
  local output = dom.pre {style = "white-space:pre-wrap"}
  local documentHelpers = js.importFromSpace("Library/Paperwork/Scripts/Documents.js")
  documentHelpers.watchEdits(text)
  documentHelpers.watchEdits(assignedDate); documentHelpers.watchEdits(assignedTitle)
  local savedDate = clientStore.get(draftKey .. ".date")
  local savedTitle = clientStore.get(draftKey .. ".title")
  if savedDate then assignedDate.value = savedDate; documentHelpers.markDirty(assignedDate) end
  if savedTitle then assignedTitle.value = savedTitle; documentHelpers.markDirty(assignedTitle) end
  assignedDate.addEventListener("input", function() clientStore.set(draftKey .. ".date", tostring(assignedDate.value or "")) end)
  assignedTitle.addEventListener("input", function() clientStore.set(draftKey .. ".title", tostring(assignedTitle.value or "")) end)
  if savedDraft and savedDraft ~= "" then documentHelpers.markDirty(text) end
  text.addEventListener("input", function()
    local ok, result = pcall(clientStore.set, draftKey, tostring(text.value or ""))
    if not ok then output.textContent = "Não foi possível guardar o rascunho local. Copie o texto antes de fechar." end
  end)
  local reviewedRun = nil
  local duplicateAck
  local duplicatePanel
  local workbench = dom.div {style = "display:flex;flex-wrap:wrap;gap:1em"}
  local original = "/.proxy/127.0.0.1:8089/v1/paperwork/documents/" .. document.id .. "/original"
  local loaded, bytes = pcall(paperwork.call, "documents/" .. document.id .. "/original")
  if loaded then
    local helpers = js.importFromSpace("Library/Paperwork/Scripts/Documents.js")
    local ok, url = pcall(helpers.originalUrl, js.tojs(bytes), document.name)
    if ok then original = url; sourceLink.href = url else output.textContent = tostring(url) end
  else output.textContent = tostring(bytes) end
  local viewer
  if tostring(document.name):lower():match("%.pdf$") then
    viewer = dom.iframe {src = original .. "#page=" .. tostring(startPage or 1), title = "Documento original", style = "flex:1 1 320px;height:65vh;border:1px solid GrayText"}
  else
    viewer = dom.img {src = original, alt = "Documento original", style = "flex:1 1 320px;max-width:100%;max-height:65vh;object-fit:contain"}
  end
  workbench.appendChild(viewer)
  local editing = dom.div {style = "flex:1 1 320px"}
  editing.appendChild(text); workbench.appendChild(editing); root.appendChild(workbench)
  local tree = dom.div {}
  local function showProposal(result)
    reviewedRun = result.run_id
    documentHelpers.markReviewed(text)
    documentHelpers.markReviewed(assignedDate); documentHelpers.markReviewed(assignedTitle)
    tree.replaceChildren()
    if duplicateAck then duplicateAck.checked = false end
    if duplicatePanel then duplicatePanel.setAttribute("style", #(result.possible_duplicates or {}) > 0 and "display:block" or "display:none") end
    for _, duplicate in ipairs(result.possible_duplicates or {}) do
      tree.appendChild(dom.p {__rawText = "Possível duplicata: " .. duplicate.context .. " já aparece em " .. duplicate.name})
    end
    local nodes = {}
    for _, block in ipairs(result.blocks or {}) do
      local item = block
      local row = dom.div {style = "margin:.4em 0 .4em 1em;padding-left:.5em;border-left:1px solid GrayText;white-space:pre-wrap"}
      row.appendChild(dom.p {__rawText = tostring(item.context or "Sem contexto") .. " · " .. tostring(item.kind) .. " · " .. item.text})
      row.appendChild(dom.button {__rawText = "Ver página " .. tostring(item.page), onclick = function() viewer.src = original .. "#page=" .. tostring(item.page) end})
      local parent = item.parent_id and nodes[item.parent_id] or tree
      parent.appendChild(row); nodes[item.id] = row
    end
    output.textContent = tostring(#(result.blocks or {})) .. " bloco(s), " .. tostring(#(result.review or {})) .. " pendência(s) de revisão."
    for _, issue in ipairs(result.review or {}) do
      tree.appendChild(dom.p {__rawText = "Revisar página " .. tostring(issue.page) .. ", linha " .. tostring(issue.line) .. ": " .. tostring(issue.reason)})
    end
  end
  root.appendChild(tree)
  root.appendChild(dom.button {__rawText = "Interpretar e revisar", onclick = function()
    local helpers = js.importFromSpace("Library/Paperwork/Scripts/Documents.js")
    local submitted = tostring(text.value or "")
    local submittedDate = tostring(assignedDate.value or "")
    local submittedTitle = tostring(assignedTitle.value or "")
    local pages = js.tolua(helpers.splitPages(submitted))
    local body = {pages = pages}
    if tostring(assignedDate.value or "") ~= "" then body.default_date = assignedDate.value end
    if tostring(assignedTitle.value or "") ~= "" then body.note_title = assignedTitle.value end
    local ok, result = pcall(paperwork.call, "documents/" .. document.id .. "/propose", "POST", body)
    if ok and tostring(text.value or "") == submitted and tostring(assignedDate.value or "") == submittedDate and tostring(assignedTitle.value or "") == submittedTitle then
      showProposal(result); clientStore.delete(draftKey); clientStore.delete(draftKey .. ".date"); clientStore.delete(draftKey .. ".title")
    elseif ok then
      reviewedRun = nil; documentHelpers.markDirty(text)
      output.textContent = "O texto mudou durante o processamento. Interprete novamente a versão atual."
    else output.textContent = tostring(result) end
  end})
  local consent = dom.input {type = "checkbox"}
  local consentLabel = dom.label {__rawText = "Autorizo enviar este documento ao provedor externo configurado"}
  consentLabel.appendChild(consent); root.appendChild(consentLabel)
  root.appendChild(dom.button {__rawText = "Transcrever com provedor configurado", onclick = function()
    if consent.checked then
      local ok, result = pcall(paperwork.call, "documents/" .. document.id .. "/policy", "PUT", {local_only = false})
      if not ok then output.textContent = tostring(result); return end
    end
    local ok, result = pcall(paperwork.call, "documents/" .. document.id .. "/transcribe", "POST", {consent_external = consent.checked})
    output.textContent = ok and js.window.JSON.stringify(js.tojs(result), nil, 2) or tostring(result)
  end})
  root.appendChild(dom.button {__rawText = "Carregar proposta existente", onclick = function()
    if documentHelpers.hasUnreviewedEdits(text) and tostring(text.value or "") ~= "" then
      output.textContent = "Há alterações locais. Interprete ou guarde o texto antes de carregar outra versão."; return
    end
    local ok, result = pcall(paperwork.call, "documents/" .. document.id .. "/proposal")
    if ok then
      showProposal(result)
      assignedDate.value = ""; assignedTitle.value = ""
      if result.assigned_context then
        assignedDate.value = result.assigned_context:match("^Daily/(.+)$") or ""
        assignedTitle.value = result.assigned_context:match("^Page/(.+)$") or ""
      end
      if result.pages then
        local helpers = js.importFromSpace("Library/Paperwork/Scripts/Documents.js")
        text.value = helpers.joinPages(js.tojs(result.pages))
      end
    else output.textContent = tostring(result) end
  end})
  root.appendChild(dom.button {__rawText = "Consultar processamento", onclick = function()
    local ok, result = pcall(paperwork.call, "jobs")
    output.textContent = ok and js.window.JSON.stringify(js.tojs(result), nil, 2) or tostring(result)
  end})
  root.appendChild(output)
  local reviewed = dom.input {type = "checkbox"}
  local label = dom.label {__rawText = "Revisei a transcrição e a estrutura propostas"}
  label.appendChild(reviewed); root.appendChild(label)
  duplicatePanel = dom.div {style = "display:none"}
  duplicateAck = dom.input {type = "checkbox"}
  local duplicateLabel = dom.label {__rawText = "Conferi a possível duplicata e quero incorporar como outro registro"}
  duplicateLabel.appendChild(duplicateAck); duplicatePanel.appendChild(duplicateLabel); root.appendChild(duplicatePanel)
  root.appendChild(dom.button {class = "pw-primary", __rawText = "Salvar nas notas", onclick = function()
    if not reviewed.checked or not reviewedRun or (documentHelpers.hasUnreviewedEdits(text) or documentHelpers.hasUnreviewedEdits(assignedDate) or documentHelpers.hasUnreviewedEdits(assignedTitle)) then output.textContent = "Interprete o texto atual e confirme a revisão antes de incorporar."; return end
    local ok, result = pcall(paperwork.call, "documents/" .. document.id .. "/incorporate", "POST", {reviewed = true, run_id = reviewedRun, allow_duplicates = duplicateAck.checked})
    output.textContent = ok and js.window.JSON.stringify(js.tojs(result), nil, 2) or tostring(result)
  end})
  root.appendChild(dom.button {__rawText = "Exportar tarefas revisadas ao Vikunja", onclick = function()
    if not reviewed.checked or not reviewedRun or (documentHelpers.hasUnreviewedEdits(text) or documentHelpers.hasUnreviewedEdits(assignedDate) or documentHelpers.hasUnreviewedEdits(assignedTitle)) then output.textContent = "Interprete o texto atual e confirme a revisão antes de exportar."; return end
    local project = tonumber(editor.prompt("ID do projeto Vikunja:"))
    if not project then return end
    local ok, result = pcall(paperwork.call, "documents/" .. document.id .. "/tasks/export", "POST", {reviewed = true, run_id = reviewedRun, project_id = project, allow_duplicates = duplicateAck.checked})
    output.textContent = ok and js.window.JSON.stringify(js.tojs(result), nil, 2) or tostring(result)
  end})
  root.appendChild(dom.button {__rawText = "Voltar à Inbox", onclick = paperwork.inbox})
  js.importFromSpace("Library/Paperwork/Scripts/Documents.js").stylePanel(root)
  editor.showPanel("modal", "5vh 3vw", root)
end
function paperwork.openDocument(identity, page)
  local document = paperwork.call("documents/" .. identity)
  paperwork.review(document, page)
end
function paperwork.source(identity, page)
  return widget.htmlBlock(dom.button {__rawText = "p. " .. tostring(page or 1), onclick = function() paperwork.openDocument(identity, page) end})
end
function paperwork.settings()
  local ok, values = pcall(paperwork.call, "settings")
  if not ok then editor.flashNotification(tostring(values), "error"); return end
  local root = dom.div {style = "padding:1em;background:Canvas;color:CanvasText;max-height:85vh;overflow:auto"}
  root.appendChild(dom.h2 {__rawText = "Paperwork — Configurações"})
  if values.shared_connections then root.appendChild(dom.p {__rawText = "Conexões e fuso compartilhados: altere-os em Relay: Configurações."}) end
  local fields = {}
  local provider = dom.select {["aria-label"] = "Provedor de transcrição", style = "display:block;width:95%"}
  for _, entry in ipairs({{"manual", "Sem IA — transcrição manual"}, {"gemini", "Google Gemini"}, {"claude", "Anthropic Claude"},
    {"openai", "ChatGPT / OpenAI (API)"}, {"openai-compatible", "Outro provedor compatível com OpenAI"}, {"local-compatible", "Modelo local / Ollama compatível"}}) do
    provider.appendChild(dom.option {value = entry[1], __rawText = entry[2]})
  end
  provider.value = values.provider or "manual"
  root.appendChild(dom.p {__rawText = "Escolha a IA"}); root.appendChild(provider); fields.provider = provider
  root.appendChild(dom.p {__rawText = "As opções usam APIs e suas próprias chaves. Informe um modelo que aceite imagem/PDF. A seleção não envia nenhum documento."})
  for _, spec in ipairs({{"timezone", "Fuso horário"}, {"silverbullet_url", "Servidor SilverBullet"}, {"vikunja_url", "API Vikunja"},
    {"model", "Identificador do modelo"}, {"local_url", "Endpoint /v1 (somente modelos locais e outros compatíveis)"}}) do
    local field = dom.input {value = tostring(values[spec[1]] or ""), disabled = values.shared_connections and (spec[1] == "silverbullet_url" or spec[1] == "vikunja_url" or spec[1] == "timezone"), style = "display:block;width:95%"}
    root.appendChild(dom.p {__rawText = spec[2]}); root.appendChild(field); fields[spec[1]] = field
  end
  local profiles = values.profiles or {}
  local previousProvider = provider.value
  provider.addEventListener("change", function()
    profiles[previousProvider] = {model = tostring(fields.model.value or ""), url = tostring(fields.local_url.value or "")}
    previousProvider = tostring(provider.value)
    local profile = profiles[previousProvider] or {}
    fields.model.value = profile.model or ""
    fields.local_url.value = profile.url or "http://127.0.0.1:11434/v1"
  end)
  local external = dom.input {type = "checkbox", checked = values.external_allowed}
  local label = dom.label {__rawText = "Permitir provedores externos (cada envio também exige confirmação)"}
  label.appendChild(external); root.appendChild(label)
  local status = dom.p {role = "status"}
  root.appendChild(dom.button {__rawText = "Salvar", onclick = function()
    local body = {external_allowed = external.checked}
    for name, field in pairs(fields) do body[name] = tostring(field.value or "") end
    local ok, result = pcall(paperwork.call, "settings", "PUT", body)
    status.textContent = ok and "Configurações salvas." or tostring(result)
  end})
  for _, name in ipairs({"silverbullet_token", "vikunja_token", "zotero_key", "gemini_key", "claude_key", "openai_key", "compatible_key", "local_key"}) do
    local key = name
    if not values.shared_connections or (key ~= "silverbullet_token" and key ~= "vikunja_token" and key ~= "zotero_key") then
    local field = dom.input {type = "password", autocomplete = "off", placeholder = key}
    root.appendChild(field)
    root.appendChild(dom.button {__rawText = "Salvar " .. key, onclick = function()
      local ok, result = pcall(paperwork.call, "settings/secrets/" .. key, "PUT", {value = tostring(field.value or "")})
      if ok then field.value = "" end
      status.textContent = ok and "Credencial salva no servidor." or tostring(result)
    end})
    end
  end
  root.appendChild(status)
  root.appendChild(dom.button {__rawText = "Fechar", onclick = function() editor.hidePanel("modal") end})
  js.importFromSpace("Library/Paperwork/Scripts/Documents.js").stylePanel(root)
  editor.showPanel("modal", "5vh 3vw", root)
end
function paperwork.importZoteroIndex()
  local root = dom.div {style = "padding:1em;background:Canvas;color:CanvasText"}
  root.appendChild(dom.h2 {__rawText = "Importar identificadores do Zotero"})
  root.appendChild(dom.p {__rawText = "No Zotero, selecione até 200 itens e use Paperwork: copiar índice selecionado. Cole o JSON aqui. O índice contém aliases e identidades, sem chaves de API. Conflitos bloqueiam todo o lote."})
  local text = dom.textarea {style = "width:95%;height:40vh", ["aria-label"] = "Índice JSON do Zotero"}
  root.appendChild(text)
  local status = dom.p {role = "status"}
  root.appendChild(dom.button {__rawText = "Importar identificadores", onclick = function()
    local parsed, body = pcall(js.window.JSON.parse, tostring(text.value or ""))
    if not parsed then status.textContent = "JSON inválido."; return end
    local ok, result = pcall(paperwork.call, "references/index", "POST", js.tolua(body))
    status.textContent = ok and "Identificadores importados. Use Ƶ alias no papel e Abra Literature Note para associar a página bibliográfica." or tostring(result)
  end})
  root.appendChild(status)
  root.appendChild(dom.button {__rawText = "Fechar", onclick = function() editor.hidePanel("modal") end})
  js.importFromSpace("Library/Paperwork/Scripts/Documents.js").stylePanel(root)
  editor.showPanel("modal", "5vh 3vw", root)
end
function paperwork.registerReference()
  local currentPage = editor.getCurrentPage()
  local front = editor.getText():match("^%-%-%-\n(.-)\n%-%-%-") or ""
  local detectedType, detectedLibrary = front:match([=[zoteroLibrary:%s*["']?(%w+):(%d+)]=])
  local detectedKey = front:match([=[zoteroKey:%s*["']?([A-Z0-9]+)]=])
  local alias = editor.prompt("Alias para escrever no papel, sem Ƶ (ex.: Silva24-Mem):")
  if not alias then return end
  local libraryType, libraryId, key, path
  if detectedType and detectedKey then
    libraryType = detectedType == "user" and "users" or detectedType == "group" and "groups" or nil
    libraryId = tonumber(detectedLibrary); key = detectedKey; path = currentPage
  else
    libraryType = editor.prompt("Tipo de biblioteca: users ou groups", "users")
    if not libraryType then return end
    libraryId = tonumber(editor.prompt("ID numérico da biblioteca Zotero:"))
    if not libraryId then return end
    key = editor.prompt("Item key Zotero (8 caracteres):")
    if not key then return end
    path = editor.prompt("Literature Note existente (opcional, sem .md):", "")
  end
  local body = {alias = alias, library_type = libraryType, library_id = libraryId, item_key = key}
  if path and path ~= "" then body.note_path = path end
  local ok, result = pcall(paperwork.call, "references", "POST", body)
  editor.flashNotification(ok and "Alias registrado. Use Ƶ " .. alias .. " no caderno." or tostring(result), ok and "info" or "error")
end
function paperwork.importReference()
  local alias = editor.prompt("Alias Zotero a importar ou abrir:")
  if not alias then return end
  local ok, result = pcall(paperwork.call, "references/" .. js.window.encodeURIComponent(alias) .. "/import", "POST")
  if ok then editor.navigate(result.path) else editor.flashNotification(tostring(result), "error") end
end
function paperwork.tasks()
  local rows = paperwork.call("tasks")
  local root = dom.div {style = "padding:1em;background:Canvas;color:CanvasText;max-height:80vh;overflow:auto"}
  root.appendChild(dom.h2 {__rawText = "Paperwork — tarefas locais"})
  local status = dom.p {role = "status"}
  for _, row in ipairs(rows) do
    local task = row
    local checkbox = dom.input {type = "checkbox", checked = task.done, ["data-paperwork-task"] = task.task_id}
    checkbox.onchange = function()
      local ok, result = pcall(paperwork.call, "tasks/" .. task.task_id, "PATCH", {done = checkbox.checked})
      status.textContent = ok and "Estado local salvo. Exporte novamente o lote para reconciliar com o Vikunja." or tostring(result)
    end
    local label = dom.label {__rawText = task.task_id .. " · " .. task.title .. " · " .. task.phase}
    label.appendChild(checkbox); root.appendChild(label); root.appendChild(dom.br {})
  end
  local helpers = js.importFromSpace("Library/Paperwork/Scripts/Documents.js")
  local printButton = dom.button {__rawText = "Imprimir tarefas abertas"}
  helpers.bindPrint(printButton, js.tojs(rows), helpers.localDate(), root)
  root.appendChild(printButton); root.appendChild(status)
  root.appendChild(dom.button {__rawText = "Atualizar", onclick = paperwork.tasks})
  root.appendChild(dom.button {__rawText = "Fechar", onclick = function() editor.hidePanel("modal") end})
  js.importFromSpace("Library/Paperwork/Scripts/Documents.js").stylePanel(root)
  editor.showPanel("modal", "5vh 3vw", root)
end
command.define {name = "Paperwork: Tarefas locais", run = paperwork.tasks}
function paperwork.search()
  local term = editor.prompt("Buscar nas transcrições:")
  if not term or term == "" then return end
  local rows = paperwork.call("search?q=" .. js.window.encodeURIComponent(term))
  local documents = paperwork.call("documents")
  local byId = {}
  for _, document in ipairs(documents) do byId[document.id] = document end
  local options = {}
  for _, row in ipairs(rows) do
    local document = byId[row.document]
    if document then table.insert(options, {name = document.name .. " · " .. row.excerpt, value = document}) end
  end
  local selected = editor.filterBox("Paperwork — resultados", options)
  if selected then paperwork.review(selected.value) end
end
command.define {name = "Paperwork: Buscar documentos", run = paperwork.search}
function paperwork.backup()
  local name = editor.prompt("Nome do snapshot (letras, números e hífen):")
  if not name then return end
  editor.flashNotification("Preparando backup de originais, banco e notas…", "info")
  local ok, result = pcall(paperwork.call, "backups", "POST", {name = name})
  editor.flashNotification(ok and "Backup concluído e verificado no destino do servidor." or tostring(result), ok and "info" or "error")
end
command.define {name = "Paperwork: Criar backup", run = paperwork.backup}
command.define {name = "Paperwork: Importar identificadores do Zotero", run = paperwork.importZoteroIndex}
command.define {name = "Paperwork: Registrar referência", run = paperwork.registerReference}
command.define {name = "Paperwork: Abrir Literature Note", run = paperwork.importReference}
command.define {name = "Paperwork: Configurações", run = paperwork.settings}
command.define {name = "Paperwork: Inbox", run = paperwork.inbox}
```
