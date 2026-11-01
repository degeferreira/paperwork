// Package paperwork implements the standalone Paperwork contracts without Python.
package paperwork

import (
 "fmt"
 "regexp"
 "strings"
 "time"
 "unicode"

 "github.com/google/uuid"
)

type Reference struct { Alias string `json:"alias"`; Page *int `json:"page"` }
type Block struct {
 ID string `json:"id"`; Page int `json:"page"`; Line int `json:"line"`
 Context *string `json:"context"`; Text string `json:"text"`; Kind string `json:"kind"`; ParentID *string `json:"parent_id"`
 Tags []string `json:"tags"`; Projects []string `json:"projects"`; References []Reference `json:"references"`
 Continuation *string `json:"continuation"`; People []string `json:"people"`
}
type Proposal struct { Blocks []Block `json:"blocks"`; Review []map[string]any `json:"review"`; AssignedContext string `json:"assigned_context,omitempty"`; PossibleDuplicates []map[string]any `json:"possible_duplicates,omitempty"`; RunID string `json:"run_id,omitempty"`; Pages []string `json:"pages,omitempty"` }
var namespace=uuid.MustParse("60500c63-b114-4bc1-acda-7b83d033ca97")
var dateMarker=regexp.MustCompile(`^// (\d{4}-\d{2}-\d{2})$`)
var cite=regexp.MustCompile(`Ƶ\s+([\pL\pN_-]+)(?:\s+p\.(\d+))?`)
var word=regexp.MustCompile(`^[\pL\pN_-]+`)
func ptr(s string)*string{return &s}
func str(s *string)string{if s==nil{return ""};return *s}
func splitlines(s string)[]string { s=strings.NewReplacer("\r\n","\n","\r","\n","\v","\n","\f","\n","\x1c","\n","\x1d","\n","\x1e","\n","\u0085","\n","\u2028","\n","\u2029","\n").Replace(s);if s==""{return nil};return strings.Split(strings.TrimSuffix(s,"\n"),"\n") }
func mentions(s,marker string)[]string {out:=[]string{};for i:=0;i<len(s);i++ {if i>0 {r:=[]rune(s[:i]);if !unicode.IsSpace(r[len(r)-1]){continue}};if strings.HasPrefix(s[i:],marker){if v:=word.FindString(s[i+len(marker):]);v!=""{out=append(out,v)}}};return out}
func Parse(pages []string,artifact string,initial *string) Proposal {
 p:=Proposal{Blocks:[]Block{},Review:[]map[string]any{}};context:=initial;var continuation *string
 type ancestor struct{col,index int};stack:=[]ancestor{};paragraphBreak:=false
 for pi,text:=range pages {for li,raw:=range splitlines(text) {page,line:=pi+1,li+1;value:=strings.TrimSpace(raw)
 review:=func(reason string){p.Review=append(p.Review,map[string]any{"page":page,"line":line,"reason":reason,"raw":raw})}
 if value==""{paragraphBreak=true;continue}
 if m:=dateMarker.FindStringSubmatch(value);m!=nil {if _,e:=time.Parse("2006-01-02",m[1]);e!=nil{review("invalid-date");context=nil;stack=nil;continue};context=ptr("Daily/"+m[1]);stack=nil;continuation=nil;continue}
 if value=="//end"||value=="::end"{if context!=nil&&((value=="//end"&&strings.HasPrefix(*context,"Page/"))||(value=="::end"&&strings.HasPrefix(*context,"Daily/"))){review("mismatched-segment-end");continue};context=nil;stack=nil;continuation=nil;continue}
 if strings.HasPrefix(value,":: "){context=ptr("Page/"+strings.TrimSpace(value[3:]));stack=nil;continuation=nil;continue}
 if strings.HasPrefix(value,"//continue "){continuation=ptr(strings.TrimSpace(value[11:]));stack=nil;p.Review=append(p.Review,map[string]any{"page":page,"line":line,"reason":"resolve-continuation","target":*continuation});continue}
 if strings.HasPrefix(value,"//"){review("invalid-command");continue}
 escaped:=strings.HasPrefix(value,`\`);depth:=-1;body:=value;prefix:=false
 if !escaped&&strings.HasPrefix(value,">"){n:=len(value)-len(strings.TrimLeft(value,">"));tail:=value[n:];if tail!=""&&(unicode.IsSpace([]rune(tail)[0])||strings.ContainsRune("•□×!?",[]rune(tail)[0])){prefix=true;depth=n;body=strings.TrimSpace(tail)}}
 symbol,content:="",body
 if !escaped&&body!=""{r:=[]rune(body);if strings.ContainsRune("•□×!?",r[0])||(r[0]=='-'&&len(r)>1&&unicode.IsSpace(r[1])){symbol=string(r[0]);content=strings.TrimSpace(string(r[1:]))}}
 paragraph:=false
 if symbol==""&&!prefix {if context!=nil&&(len(stack)==0||(strings.HasPrefix(*context,"Page/")&&paragraphBreak)){paragraph=true;stack=nil}else{if len(stack)>0 {if escaped{value=value[1:]};p.Blocks[stack[len(stack)-1].index].Text+="\n"+value}else{review("text-without-block")};paragraphBreak=false;continue}}
 paragraphBreak=false;indent:=len(raw)-len(strings.TrimLeft(raw," "));ambiguous:=false
 if strings.Contains(raw[:len(raw)-len(strings.TrimLeftFunc(raw,unicode.IsSpace))],"\t"){ambiguous=true;review("tab-indentation")}
 if depth<0 {if len(stack)==0||indent==0{depth=0}else if indent>stack[len(stack)-1].col{depth=len(stack)}else{depth=-1;for i,a:=range stack{if a.col==indent{depth=i}};if depth<0{depth=0;ambiguous=true;review("unmatched-indent")}}}
 if depth>len(stack){ambiguous=true;review("missing-parent")}
 if prefix&&indent>0&&len(stack)>0 {match:=-1;for i,a:=range stack{if a.col==indent{match=i}};if match>=0&&match!=depth{ambiguous=true;review("prefix-indent-conflict")}}
 if context==nil{review("missing-date-or-page")}
 var parent *Block;if depth>0&&!ambiguous&&depth<=len(stack){parent=&p.Blocks[stack[depth-1].index]}
 kind:="bullet";switch symbol{case "□":kind="task";case "×":kind="task-update";case "!":kind="idea";case "?":kind="question"};if paragraph{kind="paragraph"}
 tags,projects,people:=mentions(content,"#"),mentions(content,"@"),mentions(content,"@@");refs:=[]Reference{}
 for _,m:=range cite.FindAllStringSubmatch(content,-1){var pg *int;if m[2]!=""{var n int;fmt.Sscan(m[2],&n);pg=&n};refs=append(refs,Reference{m[1],pg})}
 var parentID *string;if parent!=nil{parentID=ptr(parent.ID);if len(tags)==0{tags=append(tags,parent.Tags...)};if len(projects)==0{projects=append(projects,parent.Projects...)};if len(people)==0{people=append(people,parent.People...)};if len(refs)==0{refs=append(refs,parent.References...)}}
 b:=Block{uuid.NewSHA1(namespace,[]byte(fmt.Sprintf("%s:%d:%d",artifact,page,line))).String(),page,line,context,content,kind,parentID,tags,projects,refs,continuation,people}
 p.Blocks=append(p.Blocks,b);if ambiguous{stack=nil}else{stack=append(stack[:depth],ancestor{indent,len(p.Blocks)-1})}
 }};return p
}
