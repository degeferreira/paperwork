---
tags: meta
---

```space-lua
-- priority: 2
function paperwork.desktopConnection()
  return clientStore.get("personal-paperwork.desktop.v1")
end
function paperwork.waitLocalPage(page)
  if not paperwork.desktopConnection() then return end
  for attempt = 1, 40 do
    if space.pageExists(page) then return end
    -- Wait for Desktop's native sync. Never copy over an unsynced local page.
    js.new(js.window.Promise, function(resolve) js.window.setTimeout(resolve, 250) end)
  end
  error("Nota criada no servidor, mas ainda não sincronizada neste Desktop. Verifique o Sync e selecione novamente; nenhuma nota local foi sobrescrita.")
end

function paperwork.navigate(page)
  paperwork.waitLocalPage(page)
  editor.navigate(page)
end

function paperwork.liveSearch(source, onSelect)
  local root = dom.div {style = "padding:1em;background:var(--modal-background-color, Canvas);color:var(--root-color, CanvasText);max-height:75vh;overflow:auto"}
  local input = dom.input {type = "search", placeholder = "Digite parte do autor ou título…", ["aria-label"] = "Buscar " .. source,
    style = "width:100%;box-sizing:border-box;padding:.7em;font-size:1em"}
  local status = dom.p {__rawText = "Carregando itens…", role = "status", ["aria-live"] = "polite"}
  local list = dom.div {}
  local version, timer, busy, closed = 0, nil, false, false
  local buttons = {}
  local cachedRows = {}
  local function close()
    closed = true
    if timer then js.window.clearTimeout(timer) end
    editor.hidePanel("modal")
  end
  local function choose(row)
    close()
    local ok, err = pcall(onSelect, row)
    if not ok then editor.flashNotification(tostring(err), "error") end
  end
  local function renderRows(rows)
    list.replaceChildren()
    buttons = {}
    for index, row in ipairs(rows) do
      local item, position = row, index
      local label = item.title or "Sem título"
      if source == "zotero" then label = table.concat(item.authors or {}, ", ") .. " · " .. tostring(item.year or "s.d.") .. " · " .. label end
      local button = dom.button {__rawText = label, type = "button",
        style = "display:block;width:100%;text-align:left;padding:.65em;margin:.25em 0;white-space:normal",
        onclick = function() choose(item) end,
        onkeydown = function(event)
          if event.key == "ArrowDown" and buttons[position + 1] then event.preventDefault(); buttons[position + 1].focus() end
          if event.key == "ArrowUp" then event.preventDefault(); if position == 1 then input.focus() else buttons[position - 1].focus() end end
        end}
      table.insert(buttons, button)
      list.appendChild(button)
    end
  end
  local load
  load = function()
    if closed or busy or not root.isConnected then return end
    local term = tostring(input.value or "")
    if #term == 1 then status.textContent = "Digite pelo menos 2 caracteres."; renderRows({}); return end
    busy = true
    local current = version
    status.textContent = "Buscando mais resultados…"
    local ok, rows = pcall(paperwork.call, source .. "/search?q=" .. js.window.encodeURIComponent(term))
    busy = false
    if closed or not root.isConnected then return end
    if current ~= version then
      if timer then js.window.clearTimeout(timer); timer = nil end
      load(); return
    end
    if not ok then status.textContent = tostring(rows); return end
    cachedRows = rows
    renderRows(rows)
    status.textContent = tostring(#rows) .. " resultado(s) · até 50 itens · busca por autor/título"
    if #rows == 0 then status.textContent = "Nenhum resultado. Tente parte do título ou outro autor." end
  end
  input.addEventListener("input", function()
    version = version + 1
    if timer then js.window.clearTimeout(timer) end
    local term = tostring(input.value or ""):lower()
    local matches = {}
    if #term ~= 1 then
      for _, row in ipairs(cachedRows) do
        local text = (row.title or "") .. " " .. table.concat(row.authors or {}, " ") .. " " .. tostring(row.year or "")
        if term == "" or text:lower():find(term, 1, true) then table.insert(matches, row) end
      end
    end
    renderRows(matches)
    status.textContent = tostring(#matches) .. " match(es) já carregados · aguardando busca na biblioteca…"
    if #term == 1 then status.textContent = "Digite pelo menos 2 caracteres." end
    timer = js.window.setTimeout(load, 350)
  end)
  input.addEventListener("keydown", function(event)
    if event.key == "ArrowDown" and buttons[1] then event.preventDefault(); buttons[1].focus() end
    if event.key == "Enter" and buttons[1] then event.preventDefault(); buttons[1].click() end
  end)
  root.addEventListener("keydown", function(event) if event.key == "Escape" then event.preventDefault(); close() end end)
  root.appendChild(dom.h3 {__rawText = "Buscar " .. source})
  root.appendChild(input); root.appendChild(status); root.appendChild(list)
  root.appendChild(dom.button {__rawText = "Fechar", onclick = close})
  editor.showPanel("modal", "5vh 3vw", root)
  local mounted
  local attempts = 0
  mounted = function()
    if closed then return end
    attempts = attempts + 1
    if not root.isConnected then
      if attempts < 20 then js.window.setTimeout(mounted, 50) end
      return
    end
    input.focus(); load()
  end
  js.window.setTimeout(mounted, 50)
end

function paperwork.zoteroSelected(item)
  local action = editor.filterBox("Referência", {
    {name = "Citar em APA + link para a nota + Referências", value = "cite"},
    {name = "Gerar ou atualizar nota de literatura", value = "import"},
    {name = "Abrir nota existente", value = "open"}
  })
  if not action then return end
  if action.value == "cite" then
    local page = editor.getCurrentPage()
    local helpers = js.importFromSpace("Library/Paperwork/Scripts/Citations.js")
    local result = paperwork.call("zotero/items/" .. item.key .. "/cite", "POST")
    paperwork.waitLocalPage(result.path)
    if editor.getCurrentPage() ~= page then error("Você mudou de nota durante a consulta. Selecione a referência novamente.") end
    -- Check reference markers before making any change to the current note.
    helpers.addReference(editor.getText(), result.identity, result.reference, result.path)
    editor.insertAtCursor(helpers.internalCitation(result.path, result.citation))
    editor.setText(helpers.addReference(editor.getText(), result.identity, result.reference, result.path))
    editor.save()
  elseif action.value == "import" then
    local result = paperwork.call("zotero/items/" .. item.key .. "/import", "POST")
    paperwork.navigate(result.path)
  elseif item.path then
    paperwork.navigate(item.path:gsub("%.md$", ""))
  else
    editor.flashNotification("Nenhuma nota criada para este item.", "info")
  end
end

function paperwork.zotero()
  paperwork.liveSearch("zotero", paperwork.zoteroSelected)
end

function paperwork.daily()
  if paperwork.desktopConnection() then
    local day = js.window.Intl.DateTimeFormat("sv-SE", {timeZone = "America/Sao_Paulo"}).format(js.new(js.window.Date))
    local page = "Daily/" .. day
    if not space.pageExists(page) then
      local ok, err = pcall(space.readPage, page)
      if not ok then
        if not tostring(err):lower():find("not found", 1, true) then error("Não foi possível verificar a Daily local; nenhuma nota foi sobrescrita.") end
        space.writePage(page, "---\ntype: daily\ndate: " .. day .. "\n---\n\n# " .. day ..
          '\n\n## Diário\n\n')
      end
    end
    editor.navigate(page)
    return
  end
  local result = paperwork.call("daily/open", "POST")
  editor.navigate(result.path)
end

function paperwork.dailyButton()
  return widget.htmlBlock(dom.button {__rawText = "Abrir Daily de hoje", onclick = function()
    local ok, err = pcall(paperwork.daily)
    if not ok then editor.flashNotification(tostring(err), "error") end
  end})
end

function paperwork.syncDaily()
  editor.save()
  local connection = paperwork.desktopConnection()
  if connection then
    local remote = net.proxyFetch(((paperwork.desktopConnection() or {}).server or config.get("relay.legacyServer") or "") .. "/.fs/" .. js.window.encodeURI(editor.getCurrentPage() .. ".md"),
      {headers = {Authorization = "Bearer " .. connection.token}})
    if remote.status ~= 200 or type(remote.body) ~= "string" then
      error("A Daily local ainda não chegou ao servidor. Aguarde a sincronização do Desktop.")
    end
    local remoteText = remote.body:gsub("\r\n", "\n")
    local localText = editor.getText():gsub("\r\n", "\n")
    if remoteText ~= localText then
      error("A Daily local ainda não chegou ao servidor. Aguarde a sincronização do Desktop; as tarefas serão processadas automaticamente após ela.")
    end
  end
  local result = paperwork.call("daily/reconcile", "POST", {path = editor.getCurrentPage() .. ".md"})
  editor.flashNotification(tostring(result.processed) .. " tarefa(s) processada(s).", "info")
end

command.define {name = "Daily: Abrir hoje", run = paperwork.daily}
command.define {name = "Vikunja: Sincronizar Daily atual", run = paperwork.syncDaily}
command.define {name = "Zotero: Buscar biblioteca", run = paperwork.zotero}
slashCommand.define {name = "cite", description = "Buscar Zotero e inserir citação APA com nota vinculada", run = paperwork.zotero}
```
