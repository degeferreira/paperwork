package paperwork

import (
 "errors"
 "fmt"
 "regexp"
 "strings"
)
var identityRE=regexp.MustCompile(`^[0-9a-f]{64}$`)
var blockRE=regexp.MustCompile(`^[0-9a-f-]{36}$`)
var dailyRE=regexp.MustCompile(`^Daily/\d{4}-\d{2}-\d{2}$`)
var typedMention=regexp.MustCompile(`(^|\s)(@@|@)([\pL\pN_-]+)`)
func Literal(v string)string{v=strings.NewReplacer(`\`,`\\`,"${","&#36;{","<","&lt;",">","&gt;").Replace(v);return strings.NewReplacer("`","\\`","~","\\~","[","\\[","]","\\]").Replace(v)}
func safePath(p string)bool{if p==""||strings.HasPrefix(p,"/")||strings.Contains(p,`\`){return false};for _,part:=range strings.Split(p,"/"){if part==""||part=="."||part==".."{return false}};return true}
func safeNote(p string)bool{return safePath(p)&&!strings.Contains(p,"${")&&!strings.ContainsAny(p,"[]`<>\n\r")}
func Render(p Proposal,identity string,aliases,taskIDs map[string]string)(map[string]string,error){
 if !identityRE.MatchString(identity){return nil,errors.New("Invalid document identity")};if len(p.Review)>0{return nil,errors.New("Resolve all review entries before incorporation")}
 groups:=map[string]string{};depths:=map[string]int{}
 for _,b:=range p.Blocks{if b.Page<1||b.Page>500{return nil,errors.New("Invalid source page")};if !blockRE.MatchString(b.ID){return nil,errors.New("Invalid block identity")};context:=str(b.Context)
 if strings.HasPrefix(context,"Page/"){title:=context[5:];if !safePath(title)||strings.Contains(title,"${")||strings.ContainsAny(title,"[]`\x00\n\r"){return nil,errors.New("Unsafe note destination")};for _,r:=range title{if r<32{return nil,errors.New("Invalid note destination")}};context="Paperwork/Pages/"+title}else if !dailyRE.MatchString(context){return nil,errors.New("Invalid note destination")}
 depth:=0;if b.ParentID!=nil{if d,ok:=depths[*b.ParentID];ok{depth=d+1}};depths[b.ID]=depth
 prefix:=map[string]string{"task":"□ ","task-update":"× ","idea":"! ","question":"? "}[b.Kind];text:=strings.ReplaceAll(Literal(b.Text),"\n","<br>")
 for _,mention:=range b.References{if note:=aliases[mention.Alias];note!=""{re:=regexp.MustCompile(`Ƶ\s+`+regexp.QuoteMeta(mention.Alias)+`([^\pL\pN_-]|$)`);text=re.ReplaceAllStringFunc(text,func(match string)string{loc:=re.FindStringSubmatch(match);return "[["+note+"|Ƶ "+mention.Alias+"]]"+loc[1]})}}
 text=typedMention.ReplaceAllStringFunc(text,func(m string)string{parts:=typedMention.FindStringSubmatch(m);folder:="Projects/";if parts[2]=="@@"{folder="People/"};return parts[1]+"[["+folder+parts[3]+"|"+parts[2]+parts[3]+"]]"})
 if id:=taskIDs[b.ID];id!=""{text+=" · `"+id+"`"};indent:="";if b.Kind!="paragraph"{indent=strings.Repeat("  ",depth)+"- "}
 groups[context]+=fmt.Sprintf("%s%s%s <!-- paperwork-block:%s --> [[Paperwork/Documents/%s|Documento]] ${paperwork.source(\"%s\", %d)}\n",indent,prefix,text,b.ID,identity,identity,b.Page)
 };return groups,nil
}
