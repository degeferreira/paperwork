"""Permanent handwritten aliases for real Zotero identities."""
import re

def setup(db):
    db.execute('''CREATE TABLE IF NOT EXISTS reference_aliases(
      alias TEXT PRIMARY KEY, library_type TEXT NOT NULL, library_id INTEGER NOT NULL,
      item_key TEXT NOT NULL, note_path TEXT, retired INTEGER NOT NULL DEFAULT 0)''')

def register(db,alias,library_type,library_id,item_key,note_path=None):
    if not re.fullmatch(r'[\w-]{1,100}',alias) or library_type not in ('users','groups') or library_id<=0 or not re.fullmatch(r'[A-Z0-9]{8}',item_key):
        raise ValueError('Invalid Zotero alias or identity')
    if note_path and (note_path.startswith('/') or any(p in ('..','.','') for p in note_path.split('/')) or any(c in note_path for c in '[]\n\r\\`<>') or '${' in note_path):
        raise ValueError('Invalid literature note path')
    setup(db)
    existing=db.execute('SELECT library_type,library_id,item_key,retired FROM reference_aliases WHERE alias=?',(alias,)).fetchone()
    if existing and (tuple(existing[:3])!=(library_type,library_id,item_key) or existing[3]):
        raise ValueError('Alias already assigned or retired; it cannot be reused')
    with db:
        db.execute('INSERT OR IGNORE INTO reference_aliases(alias,library_type,library_id,item_key,note_path) VALUES (?,?,?,?,?)',(alias,library_type,library_id,item_key,note_path))
    return {'alias':alias,'library_type':library_type,'library_id':library_id,'item_key':item_key}

def resolve(db,proposal):
    setup(db);results=[]
    for block in proposal['blocks']:
        for mention in block['references']:
            row=db.execute('SELECT library_type,library_id,item_key,note_path,retired FROM reference_aliases WHERE alias=?',(mention['alias'],)).fetchone()
            results.append({'block_id':block['id'],'alias':mention['alias'],'page':mention['page'],
                            'identity':{'library_type':row[0],'library_id':row[1],'item_key':row[2]} if row and not row[4] else None,
                            'note_path':row[3] if row and not row[4] else None})
    return results
