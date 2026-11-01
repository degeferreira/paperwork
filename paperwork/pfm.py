"""PFM proof of concept: deterministic parsing; unresolved context stays in review."""
from dataclasses import dataclass, field, asdict
from datetime import date
import re
import uuid

NS = uuid.UUID('60500c63-b114-4bc1-acda-7b83d033ca97')
DATE = re.compile(r'^// (\d{4}-\d{2}-\d{2})$')
EXPLICIT = re.compile(r'^(>+)(?=\s|[•□×!?])\s*(.*)$')
BULLET = re.compile(r'^(•|-(?=\s)|□|×|!|\?)\s*(.*)$')
CITE = re.compile(r'Ƶ\s+([\w-]+)(?:\s+p\.(\d+))?')

@dataclass
class Block:
    id: str
    page: int
    line: int
    context: str | None
    text: str
    kind: str
    parent_id: str | None
    tags: list = field(default_factory=list)
    projects: list = field(default_factory=list)
    references: list = field(default_factory=list)
    continuation: str | None = None


def parse(pages, artifact_id, initial_context=None):
    """Same artifact + coordinates => same IDs; edited OCR requires explicit review.

    Explicit depth is absolute in the current segment: > depth 1, >> depth 2.
    Indentation matches actual ancestor columns, never an invented fixed width.
    A structural conflict yields a review record and no silently guessed parent.
    """
    blocks, review, stack = [], [], []
    context = initial_context
    continuation = None
    paragraph_break = False
    for page, text in enumerate(pages,1):
        for line, raw in enumerate(text.splitlines(),1):
            value = raw.strip()
            if not value:
                paragraph_break = True
                continue
            marker = DATE.fullmatch(value)
            if marker:
                try:
                    date.fromisoformat(marker[1])
                except ValueError:
                    review.append(dict(page=page,line=line,reason='invalid-date',raw=raw))
                    context=None; stack=[]; continue
                context='Daily/'+marker[1]; stack=[]; continuation=None; continue
            if value in ('//end','::end'):
                if context and ((value=='//end' and context.startswith('Page/')) or (value=='::end' and context.startswith('Daily/'))):
                    review.append(dict(page=page,line=line,reason='mismatched-segment-end',raw=raw));continue
                context=None; stack=[]; continuation=None; continue
            if value.startswith(':: '):
                context='Page/'+value[3:].strip(); stack=[]; continuation=None; continue
            if value.startswith('//continue '):
                continuation=value[11:].strip()
                stack=[]
                review.append(dict(page=page,line=line,reason='resolve-continuation',target=continuation))
                continue
            if value.startswith('//'):
                review.append(dict(page=page,line=line,reason='invalid-command',raw=raw)); continue
            escaped = value.startswith('\\')
            prefix=None if escaped else EXPLICIT.match(value)
            depth=len(prefix[1]) if prefix else None
            body=prefix[2] if prefix else value
            bullet=None if escaped else BULLET.match(body)
            paragraph = False
            if not bullet and not prefix:
                if context and (not stack or (context.startswith('Page/') and paragraph_break)):
                    paragraph = True
                    stack = []
                else:
                    if stack:
                        stack[-1][1].text+='\n'+(value[1:] if escaped else value)
                    else:
                        review.append(dict(page=page,line=line,reason='text-without-block',raw=raw))
                    paragraph_break = False
                    continue
            paragraph_break = False
            indent=len(raw)-len(raw.lstrip(' '))
            ambiguous=False
            if '\t' in raw[:len(raw)-len(raw.lstrip())]:
                ambiguous=True
                review.append(dict(page=page,line=line,reason='tab-indentation',raw=raw))
            if depth is None:
                if not stack or indent==0:
                    depth=0
                elif indent>stack[-1][0]:
                    depth=len(stack)
                else:
                    levels=[i for i,(col,_) in enumerate(stack) if col==indent]
                    if levels:
                        depth=levels[-1]
                    else:
                        depth=0; ambiguous=True
                        review.append(dict(page=page,line=line,reason='unmatched-indent',raw=raw))
            if depth>len(stack):
                ambiguous=True
                review.append(dict(page=page,line=line,reason='missing-parent',raw=raw))
            if prefix and indent and stack:
                matches=[i for i,(col,_) in enumerate(stack) if col==indent]
                if matches and matches[-1]!=depth:
                    ambiguous=True
                    review.append(dict(page=page,line=line,reason='prefix-indent-conflict',raw=raw))
            if context is None:
                review.append(dict(page=page,line=line,reason='missing-date-or-page',raw=raw))
            parent=stack[depth-1][1] if depth and not ambiguous and depth<=len(stack) else None
            symbol, content=(bullet[1],bullet[2]) if bullet else ('paragraph' if paragraph else '•',body)
            tags=re.findall(r'(?:^|\s)#([\w-]+)',content)
            projects=re.findall(r'(?:^|\s)@([\w-]+)',content)
            refs=[dict(alias=m[1],page=int(m[2]) if m[2] else None) for m in CITE.finditer(content)]
            block=Block(str(uuid.uuid5(NS,f'{artifact_id}:{page}:{line}')),page,line,context,content,
                        {'□':'task','×':'task-update','!':'idea','?':'question','paragraph':'paragraph'}.get(symbol,'bullet'),
                        parent.id if parent else None,tags or (parent.tags[:] if parent else []),
                        projects or (parent.projects[:] if parent else []),refs or (parent.references[:] if parent else []),continuation)
            blocks.append(block)
            if ambiguous:
                stack=[]
            else:
                stack=stack[:depth]+[(indent,block)]
    return dict(blocks=[asdict(b) for b in blocks],review=review)
