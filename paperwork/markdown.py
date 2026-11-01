"""Render reviewed PFM into additive Markdown; every block keeps its identity."""
import re
from collections import defaultdict

def literal(value):
    # Imported text cannot execute Space Lua, directives or raw HTML.
    value=str(value).replace('\\','\\\\').replace('${','&#36;{').replace('<','&lt;').replace('>','&gt;')
    for symbol in ('`','~','[',']'):value=value.replace(symbol,'\\'+symbol)
    return value

def render(proposal,identity,aliases=None,task_ids=None):
    if not re.fullmatch(r'[0-9a-f]{64}',identity):raise ValueError('Invalid document identity')
    if proposal['review']:raise ValueError('Resolve all review entries before incorporation')
    groups=defaultdict(list);depths={}
    for block in proposal['blocks']:
        if not isinstance(block['page'],int) or not 1<=block['page']<=500:raise ValueError('Invalid source page')
        if not re.fullmatch(r'[0-9a-f-]{36}',block['id']):raise ValueError('Invalid block identity')
        context=block['context']
        if not context or not re.fullmatch(r'Daily/\d{4}-\d{2}-\d{2}|Page/[^\x00-\x1f]+',context):raise ValueError('Invalid note destination')
        if context.startswith('Page/'):
            title=context[5:]
            if any(p in ('.','..','') for p in title.split('/')) or title.startswith('/') or '\\' in title or '${' in title or any(c in title for c in '[]`'):
                raise ValueError('Unsafe note destination')
            context='Paperwork/Pages/'+title
        depth=depths.get(block['parent_id'],-1)+1 if block['parent_id'] else 0
        depths[block['id']]=depth
        # Paperwork tasks are not native Daily checkboxes until exported:
        # Relay's existing scanner must never create them automatically.
        prefix={'task':'□ ','task-update':'× ','idea':'! ','question':'? '}.get(block['kind'],'')
        text=literal(block['text']).replace('\n','<br>')
        for mention in block['references']:
            note=(aliases or {}).get(mention['alias'])
            if note:
                text=re.sub(r'Ƶ\s+'+re.escape(mention['alias'])+r'(?![\w-])',lambda _: '[['+note+'|Ƶ '+mention['alias']+']]',text)
        if task_ids and block['id'] in task_ids:
            text+=' · `'+task_ids[block['id']]+'`'
        marker=f'<!-- paperwork-block:{block["id"]} -->'
        link=f'[[Paperwork/Documents/{identity}|Documento]] ${{paperwork.source("{identity}", {block["page"]})}}'
        groups[context].append(('' if block['kind']=='paragraph' else '  '*depth+'- ')+prefix+text+' '+marker+' '+link)
    return {path:'\n'.join(lines)+'\n' for path,lines in groups.items()}
