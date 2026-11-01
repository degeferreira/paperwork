"""Paperwork local document and transcription tools (no network access)."""
import argparse
import json
from .store import Store

def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--library',required=True)
    sub=parser.add_subparsers(dest='command',required=True)
    imp=sub.add_parser('import');imp.add_argument('file')
    propose=sub.add_parser('propose');propose.add_argument('document');propose.add_argument('pages',nargs='+')
    search=sub.add_parser('search');search.add_argument('query')
    sub.add_parser('list')
    backup=sub.add_parser('backup');backup.add_argument('destination')
    verify=sub.add_parser('verify');verify.add_argument('destination')
    restore_parser=sub.add_parser('restore');restore_parser.add_argument('snapshot');restore_parser.add_argument('--space')
    args=parser.parse_args()
    if args.command=='restore':
        from .backup import restore
        print(json.dumps(restore(args.snapshot,args.library,args.space),ensure_ascii=False,indent=2));return
    store=Store(args.library)
    try:
        if args.command=='import': result={'document':store.import_original(args.file)}
        elif args.command=='propose':
            from pathlib import Path
            result=store.propose(args.document,[Path(p).read_text() for p in args.pages])
        elif args.command=='search': result=store.db.execute('SELECT document_id,snippet(search,1,?, ?,?,20) FROM search WHERE search MATCH ?',('[',']','…',args.query)).fetchall()
        elif args.command=='list': result=store.db.execute('SELECT id,name FROM documents').fetchall()
        elif args.command=='backup': result=store.backup(args.destination)
        elif args.command=='verify':
            from .store import verify_snapshot
            result=verify_snapshot(args.destination)
        print(json.dumps(result,ensure_ascii=False,indent=2))
    finally:store.db.close()

if __name__=='__main__':main()
