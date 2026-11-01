package paperwork

import (
 "context"
 "errors"
 "os"
 "os/exec"
 "path/filepath"
 "regexp"
 "strconv"
 "time"
)
func runPDF(ctx context.Context,args ...string)([]byte,error){if _,e:=exec.LookPath(args[0]);e!=nil{return nil,errors.New("Install poppler-utils for local PDF transcription")};if _,e:=exec.LookPath("prlimit");e!=nil{return nil,errors.New("Install util-linux for bounded PDF processing")};ctx,cancel:=context.WithTimeout(ctx,60*time.Second);defer cancel();cmd:=exec.CommandContext(ctx,"prlimit",append([]string{"--as=134217728","--cpu=45","--"},args...)...);return cmd.Output()}
func PDFPageCount(ctx context.Context,path string)(int,error){data,e:=runPDF(ctx,"pdfinfo",path);if e!=nil{return 0,e};m:=regexp.MustCompile(`(?m)^Pages:\s+(\d+)`).FindSubmatch(data);if m==nil{return 0,errors.New("Unsupported PDF page count")};n,e:=strconv.Atoi(string(m[1]));if e!=nil||n<1||n>500{return 0,errors.New("Unsupported PDF page count")};return n,nil}
func RenderPDFPage(ctx context.Context,path string,page int,destination string)([]byte,error){if page<1||page>500{return nil,errors.New("Invalid source page")};_,e:=runPDF(ctx,"pdftoppm","-f",strconv.Itoa(page),"-l",strconv.Itoa(page),"-singlefile","-scale-to","1800","-png",path,destination);if e!=nil{return nil,e};output:=destination+".png";defer os.Remove(output);info,e:=os.Stat(output);if e!=nil{return nil,e};if info.Size()>12_000_000{return nil,errors.New("Rendered page too large")};return os.ReadFile(filepath.Clean(output))}
