"""Local proof: immutable originals, durable identity and verifiable snapshots."""
from contextlib import closing
import hashlib
import json
import sqlite3
import uuid
import datetime as dt
from pathlib import Path
from .pfm import parse

def file_hash(path):
    with Path(path).open('rb') as source:
        return hashlib.file_digest(source,'sha256').hexdigest()

class Store:
    def __init__(self, root):
        self.root=Path(root); self.root.mkdir(parents=True,exist_ok=True)
        self.originals=self.root/'originals'; self.originals.mkdir(exist_ok=True)
        self.db=sqlite3.connect(self.root/'paperwork.sqlite')
        version=self.db.execute('PRAGMA user_version').fetchone()[0]
        if version>1:
            self.db.close();raise ValueError('Library schema newer than this Paperwork version')
        self.db.executescript('''
        CREATE TABLE IF NOT EXISTS documents(id TEXT PRIMARY KEY, name TEXT NOT NULL);
        CREATE TABLE IF NOT EXISTS segment_signatures(signature TEXT NOT NULL,document_id TEXT NOT NULL,context TEXT NOT NULL,PRIMARY KEY(signature,document_id,context));
        CREATE TABLE IF NOT EXISTS library_meta(key TEXT PRIMARY KEY,value TEXT NOT NULL);
        CREATE TABLE IF NOT EXISTS task_data(block_id TEXT PRIMARY KEY,title TEXT NOT NULL,project_id INTEGER NOT NULL,parent_block_id TEXT,done INTEGER NOT NULL DEFAULT 0);
        CREATE TABLE IF NOT EXISTS task_mentions(block_id TEXT PRIMARY KEY,task_id TEXT NOT NULL);
        CREATE TABLE IF NOT EXISTS document_metadata(document_id TEXT PRIMARY KEY, imported_at TEXT NOT NULL, captured_at TEXT, notebook TEXT, local_only INTEGER NOT NULL DEFAULT 1);
        CREATE TABLE IF NOT EXISTS proposals(document_id TEXT PRIMARY KEY, payload TEXT NOT NULL);
        CREATE TABLE IF NOT EXISTS incorporation(document_id TEXT PRIMARY KEY, proposal_hash TEXT NOT NULL, phase TEXT NOT NULL);
        CREATE TABLE IF NOT EXISTS tasks(block_id TEXT PRIMARY KEY, task_id TEXT UNIQUE NOT NULL, instance TEXT NOT NULL, remote_id INTEGER, phase TEXT NOT NULL);
        CREATE TABLE IF NOT EXISTS processing(id TEXT PRIMARY KEY, document_id TEXT NOT NULL, state TEXT NOT NULL, detail TEXT NOT NULL);
        CREATE TABLE IF NOT EXISTS runs(id TEXT PRIMARY KEY, document_id TEXT NOT NULL, created TEXT NOT NULL, pages TEXT NOT NULL, payload TEXT NOT NULL);
        CREATE TABLE IF NOT EXISTS corrections(block_id TEXT PRIMARY KEY, text TEXT NOT NULL);
        CREATE VIRTUAL TABLE IF NOT EXISTS search USING fts5(document_id UNINDEXED, text);
        PRAGMA user_version=1;
        ''')
        with self.db:
            self.db.execute('INSERT OR IGNORE INTO library_meta VALUES (?,?)',('library_id',str(uuid.uuid4())))
            self.db.execute('INSERT OR IGNORE INTO library_meta VALUES (?,?)',('task_sequence','0'))

    def import_original(self, path):
        path=Path(path)
        digest=hashlib.sha256()
        with path.open('rb') as source:
            for chunk in iter(lambda:source.read(1024*1024),b''): digest.update(chunk)
        identity=digest.hexdigest(); target=self.originals/identity
        if not target.exists():
            import os
            import tempfile
            fd,name=tempfile.mkstemp(prefix='.pending-',dir=self.originals)
            pending=Path(name)
            try:
                with path.open('rb') as source,os.fdopen(fd,'wb') as out:
                    for chunk in iter(lambda:source.read(1024*1024),b''):out.write(chunk)
                    out.flush();os.fsync(out.fileno())
                if file_hash(pending)!=identity:raise ValueError('Original changed during import')
                try:os.link(pending,target)
                except FileExistsError:pass
            finally:pending.unlink(missing_ok=True)
            if os.name=='posix':
                directory=os.open(self.originals,os.O_RDONLY)
                try:os.fsync(directory)
                finally:os.close(directory)
        if file_hash(target)!=identity:raise ValueError('Stored original failed integrity verification')
        with self.db:
            self.db.execute('INSERT OR IGNORE INTO documents VALUES (?,?)',(identity,path.name))
            self.db.execute('INSERT OR IGNORE INTO document_metadata(document_id,imported_at) VALUES (?,?)',(identity,dt.datetime.now(dt.timezone.utc).isoformat()))
        return identity

    def propose(self, identity, pages, initial_context=None):
        if not self.db.execute('SELECT 1 FROM documents WHERE id=?',(identity,)).fetchone():
            raise ValueError('Unknown document')
        result=parse(pages,identity,initial_context)
        if initial_context:result['assigned_context']=initial_context
        from .duplicates import candidates
        duplicates=candidates(self.db,result,identity)
        if duplicates:result['possible_duplicates']=duplicates
        # Proposals never modify canonical Markdown or existing corrections.
        run_id=str(uuid.uuid4())
        with self.db:
            self.db.execute('INSERT INTO runs VALUES (?,?,?,?,?)',
                            (run_id,identity,dt.datetime.now(dt.timezone.utc).isoformat(),json.dumps(pages,ensure_ascii=False),json.dumps(result,ensure_ascii=False)))
            self.db.execute('INSERT OR REPLACE INTO proposals VALUES (?,?)',(identity,json.dumps(result,ensure_ascii=False)))
            self.db.execute('DELETE FROM search WHERE document_id=?',(identity,))
            self.db.execute('INSERT INTO search VALUES (?,?)',(identity,'\n'.join(pages)))
        return {**result,'run_id':run_id}

    def backup(self, destination):
        import shutil
        destination=Path(destination); destination.mkdir(parents=True,exist_ok=False,mode=0o700)
        with closing(sqlite3.connect(destination/'paperwork.sqlite')) as snapshot:
            self.db.backup(snapshot)
        shutil.copytree(self.originals,destination/'originals',ignore=shutil.ignore_patterns('.pending-*'))
        manifest={str(p.relative_to(destination)):file_hash(p)
                  for p in destination.rglob('*') if p.is_file()}
        (destination/'manifest.json').write_text(json.dumps(manifest,indent=2)+'\n')
        verify_snapshot(destination)
        return manifest

def verify_snapshot(destination):
    destination=Path(destination).resolve()
    entries=list(destination.rglob('*'))
    if any(p.is_symlink() for p in entries):raise ValueError('Snapshot contains symbolic links')
    manifest=json.loads((destination/'manifest.json').read_text())
    if not isinstance(manifest,dict):raise ValueError('Invalid snapshot manifest')
    actual={str(p.relative_to(destination)) for p in entries if p.is_file() and p.name!='manifest.json'}
    if actual!=set(manifest):raise ValueError('Snapshot has unlisted or missing files')
    for name,digest in manifest.items():
        path=(destination/name).resolve()
        if not path.is_relative_to(destination) or not path.is_file() or file_hash(path)!=digest:
            raise ValueError('Missing or corrupted snapshot file: '+name)
    with closing(sqlite3.connect('file:'+str(destination/'paperwork.sqlite')+'?mode=ro',uri=True)) as db:
        if db.execute('PRAGMA integrity_check').fetchone()[0]!='ok':raise ValueError('Invalid SQLite snapshot')
        if db.execute('PRAGMA user_version').fetchone()[0]!=1:raise ValueError('Unsupported snapshot schema')
        for (identity,) in db.execute('SELECT id FROM documents'):
            if manifest.get('originals/'+identity)!=identity:raise ValueError('Original does not match its canonical SHA-256 identity')
    return {'verified':True,'files':len(manifest)}

def recover_processing():
    import os
    library=Store(os.environ.get('PAPERWORK_HOME','/paperwork'))
    try:
        with library.db:
            library.db.execute("UPDATE processing SET state='interrupted' WHERE state IN ('queued','transcribed')")
    finally:library.db.close()
