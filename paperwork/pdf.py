"""Bounded page conversion using Poppler, never shell interpolation."""
import re
import shutil
import subprocess

def run(arguments):
    if not shutil.which(arguments[0]):raise ValueError('Install poppler-utils for local PDF transcription')
    limiter=shutil.which('prlimit')
    if not limiter:raise ValueError('Install util-linux for bounded PDF processing')
    command=[limiter,'--as=134217728','--cpu=45','--',*arguments]
    result=subprocess.run(command,stdout=subprocess.PIPE,stderr=subprocess.PIPE,timeout=60,check=True)
    return result.stdout

def page_count(path):
    output=run(['pdfinfo',str(path)]).decode(errors='replace')
    match=re.search(r'^Pages:\s+(\d+)',output,re.M)
    if not match or not 1<=int(match[1])<=500:raise ValueError('Unsupported PDF page count')
    return int(match[1])

def render_page(path,page,destination):
    run(['pdftoppm','-f',str(page),'-l',str(page),'-singlefile','-scale-to','1800','-png',str(path),str(destination)])
    output=destination.with_suffix('.png')
    data=output.read_bytes();output.unlink()
    if len(data)>12_000_000:raise ValueError('Rendered page too large')
    return data
