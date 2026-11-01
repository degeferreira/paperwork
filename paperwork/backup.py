"""Optimistic consistent export: refuse completion if native notes changed."""
import json
from pathlib import Path
from .store import file_hash,verify_snapshot

async def backup_with_notes(library,space,destination):
    destination=Path(destination)
    initial=sorted(p for p in await space.files() if p.endswith('.md') and not p.startswith('Library/'))
    revisions={}
    library.backup(destination)
    (destination/'manifest.json').unlink()
    for path in initial:
        relative=Path(path)
        if relative.is_absolute() or '..' in relative.parts or '\\' in path:raise ValueError('Unsafe note path')
        content,etag=await space.read(path)
        if content is None or not etag:raise ValueError('Cannot establish note revision')
        target=destination/'notes'/relative;target.parent.mkdir(parents=True,exist_ok=True)
        target.write_text(content);revisions[path]=etag
    for path,etag in revisions.items():
        _,current=await space.read(path)
        if current!=etag:raise ValueError('Notes changed during backup; retry with a new destination')
    final=sorted(p for p in await space.files() if p.endswith('.md') and not p.startswith('Library/'))
    if final!=initial:raise ValueError('Note list changed during backup')
    manifest={str(p.relative_to(destination)):file_hash(p) for p in destination.rglob('*') if p.is_file()}
    (destination/'manifest.json').write_text(json.dumps(manifest,indent=2)+'\n')
    return verify_snapshot(destination)

def restore(snapshot,destination,space_destination=None):
    import os
    import shutil
    import tempfile
    snapshot=Path(snapshot).resolve();destination=Path(destination).resolve()
    verify_snapshot(snapshot)
    if destination.exists():raise ValueError('Restore destination must be new')
    space_destination=Path(space_destination).resolve() if space_destination else None
    if space_destination and space_destination.exists():raise ValueError('Space destination must be new')
    destination.parent.mkdir(parents=True,exist_ok=True)
    with tempfile.TemporaryDirectory(prefix='.paperwork-restore-',dir=destination.parent) as temporary:
        stage=Path(temporary)/'library'
        shutil.copytree(snapshot,stage)
        verify_snapshot(stage)
        notes=stage/'notes'
        if space_destination and notes.exists():
            space_destination.parent.mkdir(parents=True,exist_ok=True)
            shutil.copytree(notes,space_destination)
        # The source snapshot remains the evidence. Runtime notes live in the Space.
        if notes.exists() and space_destination:shutil.rmtree(notes)
        (stage/'manifest.json').unlink()
        os.replace(stage,destination)
    return {'restored':True,'library':str(destination),'space':str(space_destination) if space_destination else None}
