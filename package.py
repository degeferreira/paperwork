"""Build independent Paperwork installation, with no Relay or private code."""
import argparse
import json
import shutil
from pathlib import Path
SOURCE=Path(__file__).resolve().parent

def build(destination):
    destination=Path(destination).resolve()
    if destination.exists():raise ValueError('Destination exists')
    destination.mkdir(parents=True)
    source=SOURCE
    shutil.copytree(source/'paperwork',destination/'backend/paperwork',ignore=shutil.ignore_patterns('__pycache__','*.pyc'))
    for name in ('Dockerfile','requirements.txt'):shutil.copy2(source/name,destination/'backend'/name)
    shutil.copytree(source/'Library',destination/'libraries/Paperwork')
    shutil.copy2(source/'config.example.json',destination/'config.example.json')
    compose=(source/'compose.yaml').read_text()
    (destination/'compose.yaml').write_text(compose)
    (destination/'package.json').write_text(json.dumps({'version':(SOURCE/'VERSION').read_text().strip(),'kind':'paperwork','silverbullet':'2.11.1'},indent=2)+'\n')
    return destination
if __name__=='__main__':
    cli=argparse.ArgumentParser(description=__doc__);cli.add_argument('destination')
    print(build(cli.parse_args().destination))
