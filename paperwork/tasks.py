"""Manual reviewed-task export; uncertain creates are never blindly repeated."""
import re

def ensure_local(proposal,library):
    if proposal['review']:raise ValueError('Resolve review entries first')
    result={}
    for block in proposal['blocks']:
        if block['kind']=='task-update':
            identity=block['text'].split()[0] if block['text'].split() else ''
            row=library.db.execute('SELECT block_id FROM tasks WHERE task_id=?',(identity,)).fetchone()
            if not row:raise ValueError('Unknown task update ID: '+identity)
            with library.db:
                library.db.execute('UPDATE task_data SET done=1 WHERE block_id=?',(row[0],))
                library.db.execute('UPDATE tasks SET phase=? WHERE task_id=?',('pending-update',identity))
                library.db.execute('INSERT OR IGNORE INTO task_mentions VALUES (?,?)',(block['id'],identity))
            continue
        if block['kind']!='task':continue
        first=block['text'].split()[0] if block['text'].split() else ''
        mentioned=library.db.execute('SELECT task_id FROM tasks WHERE task_id=?',(first,)).fetchone()
        if mentioned:
            result[block['id']]=mentioned[0]
            with library.db:library.db.execute('INSERT OR IGNORE INTO task_mentions VALUES (?,?)',(block['id'],mentioned[0]))
            continue
        if re.fullmatch(r'T\d{5,}',first):raise ValueError('Unknown printed task ID')
        existing=library.db.execute('SELECT task_id FROM tasks WHERE block_id=?',(block['id'],)).fetchone()
        if existing:result[block['id']]=existing[0];continue
        with library.db:
            sequence=int(library.db.execute("SELECT value FROM library_meta WHERE key='task_sequence'").fetchone()[0])+1
            identity=f'T{sequence:05d}'
            library.db.execute("UPDATE library_meta SET value=? WHERE key='task_sequence'",(str(sequence),))
            library.db.execute('INSERT INTO tasks VALUES (?,?,?,?,?)',(block['id'],identity,'',None,'local'))
            library.db.execute('INSERT INTO task_data VALUES (?,?,?,?,?)',(block['id'],block['text'],0,block['parent_id'],0))
        result[block['id']]=identity
    return result

async def export(proposal,library,connector,instance,project):
    if proposal['review']:raise ValueError('Resolve review entries first')
    ensure_local(proposal,library)
    exported={};results=[]
    for block in proposal['blocks']:
        if block['kind']!='task':continue
        first=block['text'].split()[0] if block['text'].split() else ''
        mentioned=library.db.execute('SELECT task_id,instance,remote_id,phase FROM tasks WHERE task_id=?',(first,)).fetchone()
        if mentioned:
            if mentioned[1] and mentioned[1]!=instance:raise ValueError('Referenced task belongs to another instance')
            if not mentioned[2]:raise ValueError('Referenced task must be exported from its original record first')
            with library.db:library.db.execute('INSERT OR IGNORE INTO task_mentions VALUES (?,?)',(block['id'],mentioned[0]))
            exported[block['id']]=mentioned[2]
            results.append({'task_id':mentioned[0],'block_id':block['id'],'remote_id':mentioned[2]})
            continue
        if re.fullmatch(r'T\d{5,}',first):raise ValueError('Unknown printed task ID; review before exporting')
        existing=library.db.execute('SELECT task_id,instance,remote_id,phase FROM tasks WHERE block_id=?',(block['id'],)).fetchone()
        if existing and existing[1] and existing[1]!=instance:raise ValueError('Task belongs to another Vikunja instance')
        if existing:
            identity=existing[0]
        else:
            with library.db:
                sequence=int(library.db.execute("SELECT value FROM library_meta WHERE key='task_sequence'").fetchone()[0])+1
                identity=f'T{sequence:05d}'
                library.db.execute("UPDATE library_meta SET value=? WHERE key='task_sequence'",(str(sequence),))
                library.db.execute('INSERT INTO tasks VALUES (?,?,?,?,?)',(block['id'],identity,instance,None,'prepared'))
                library.db.execute('INSERT INTO task_data VALUES (?,?,?,?,?)',(block['id'],block['text'],project,block['parent_id'],0))
        with library.db:
            library.db.execute('UPDATE tasks SET instance=? WHERE block_id=?',(instance,block['id']))
            library.db.execute('UPDATE task_data SET project_id=? WHERE block_id=?',(project,block['id']))
        done=bool(library.db.execute('SELECT done FROM task_data WHERE block_id=?',(block['id'],)).fetchone()[0])
        remote=existing[2] if existing else None
        namespace=library.db.execute("SELECT value FROM library_meta WHERE key='library_id'").fetchone()[0]
        marker='Paperwork origin: '+namespace+':'+identity
        if not remote:
            candidates=await connector.paginated_vk('/tasks',{'s':identity})
            matches=[r for r in candidates if marker in (r.get('description') or '')]
            if len(matches)>1:raise ValueError('Duplicate remote identity; manual review required')
            if matches:remote=matches[0]['id']
            elif existing and existing[3]=='uncertain':
                raise ValueError('Previous creation uncertain; verify remotely before retrying')
            else:
                # Durable uncertainty before sending the only create request.
                with library.db:library.db.execute('UPDATE tasks SET phase=? WHERE block_id=?',('uncertain',block['id']))
                response=await connector.vk('PUT',f'/projects/{project}/tasks',json={'title':block['text'],'description':marker,'done':done},retries=False)
                remote=response.json()['id']
            with library.db:library.db.execute('UPDATE tasks SET remote_id=?,phase=? WHERE block_id=?',(remote,'synced',block['id']))
        if done or (existing and existing[3]=='pending-update'):
            current=(await connector.vk('GET',f'/tasks/{remote}')).json()
            if bool(current.get('done'))!=done:
                current['done']=done
                await connector.vk('POST',f'/tasks/{remote}',json=current,retries=False)
            with library.db:library.db.execute('UPDATE tasks SET phase=? WHERE block_id=?',('completed' if done else 'synced',block['id']))
        exported[block['id']]=remote
        parent=exported.get(block['parent_id'])
        if parent:
            current=(await connector.vk('GET',f'/tasks/{parent}')).json()
            related=current.get('related_tasks',{}).get('subtask',[]) or []
            if not any(t['id']==remote for t in related):
                await connector.vk('PUT',f'/tasks/{parent}/relations',json={'other_task_id':remote,'relation_kind':'subtask'},retries=False)
        results.append({'task_id':identity,'block_id':block['id'],'remote_id':remote})
    return results

async def apply_updates(proposal,library,connector,instance):
    results=[]
    for block in proposal['blocks']:
        if block['kind']!='task-update':continue
        identity=block['text'].split()[0] if block['text'].split() else ''
        row=library.db.execute('SELECT instance,remote_id,phase FROM tasks WHERE task_id=?',(identity,)).fetchone()
        if not row or (row[0] and row[0]!=instance):raise ValueError('Unknown task update or instance mismatch: '+identity)
        with library.db:library.db.execute('UPDATE task_data SET done=1 WHERE block_id=(SELECT block_id FROM tasks WHERE task_id=?)',(identity,))
        if not row[1]:
            results.append({'task_id':identity,'done':True,'remote_pending':True});continue
        task=(await connector.vk('GET',f'/tasks/{row[1]}')).json()
        if not task.get('done'):
            task['done']=True
            await connector.vk('POST',f'/tasks/{row[1]}',json=task,retries=False)
        with library.db:
            library.db.execute('UPDATE task_data SET done=1 WHERE block_id=(SELECT block_id FROM tasks WHERE task_id=?)',(identity,))
            library.db.execute('UPDATE tasks SET phase=? WHERE task_id=?',('completed',identity))
        results.append({'task_id':identity,'done':True})
    return results
