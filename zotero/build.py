"""Build an installable Zotero XPI without runtime, credentials or dependencies."""
from pathlib import Path
import hashlib,json,zipfile
root=Path(__file__).resolve().parent;target=root/'dist';target.mkdir(exist_ok=True)
version=json.loads((root/'manifest.json').read_text())['version']
xpi=target/f'paperwork-zotero-{version}.xpi'
with zipfile.ZipFile(xpi,'w',zipfile.ZIP_DEFLATED) as out:
    for name in ('manifest.json','bootstrap.js','aliases.js'):out.write(root/name,name)
(target/(xpi.name+'.sha256')).write_text(hashlib.sha256(xpi.read_bytes()).hexdigest()+'  '+xpi.name+'\n')
print(xpi)
