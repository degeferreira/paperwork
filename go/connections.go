package paperwork

import (
 "bytes"
 "context"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "net/http"
 "net/url"
 "os"
 "strconv"
 "strings"
 "time"
)
type UpstreamError struct{Status int}
func(e *UpstreamError)Error()string{return fmt.Sprintf("Upstream connection unavailable (status %d)",e.Status)}
func NewHTTPClient(timeout time.Duration)*http.Client{return &http.Client{Timeout:timeout,CheckRedirect:func(*http.Request,[]*http.Request)error{return http.ErrUseLastResponse}}}
func request(ctx context.Context,client *http.Client,method,endpoint string,headers map[string]string,body []byte)([]byte,http.Header,error){req,e:=http.NewRequestWithContext(ctx,method,endpoint,bytes.NewReader(body));if e!=nil{return nil,nil,e};for k,v:=range headers{req.Header.Set(k,v)};res,e:=client.Do(req);if e!=nil{return nil,nil,&UpstreamError{0}};defer res.Body.Close();if res.StatusCode<200||res.StatusCode>=300{return nil,res.Header,&UpstreamError{res.StatusCode}};data,e:=io.ReadAll(io.LimitReader(res.Body,64_000_001));if e!=nil{return nil,nil,&UpstreamError{0}};if len(data)>64_000_000{return nil,nil,errors.New("Upstream response too large")};return data,res.Header,nil}
func credential(path string)(string,error){data,e:=os.ReadFile(path);if e!=nil{return "",e};token:=strings.TrimSpace(string(data));if token==""{return "",errors.New("Credential not configured")};return token,nil}
type Space interface{Read(context.Context,string)(*string,string,error);Write(context.Context,string,string,string)error;Files(context.Context)([]string,error)}
type SpaceConnection struct{URL,TokenFile string;Client *http.Client}
func NewSpaceConnection()(*SpaceConnection,error){config,e:=Configuration();if e!=nil{return nil,e};endpoint,e:=ServerURL(stringValue(config["silverbullet_url"]));if e!=nil{return nil,e};path:=stringValue(config["silverbullet_token_file"]);if _,e=credential(path);e!=nil{return nil,e};return &SpaceConnection{endpoint,path,NewHTTPClient(30*time.Second)},nil}
func(s *SpaceConnection)headers()(map[string]string,error){token,e:=credential(s.TokenFile);return map[string]string{"Authorization":"Bearer "+token,"X-Sync-Mode":"true"},e}
func(s *SpaceConnection)path(path string)(string,error){if !safePath(path){return "",errors.New("Unsafe path")};parts:=strings.Split(path,"/");for i,p:=range parts{parts[i]=url.PathEscape(p)};return s.URL+"/.fs/"+strings.Join(parts,"/"),nil}
func(s *SpaceConnection)Read(ctx context.Context,path string)(*string,string,error){endpoint,e:=s.path(path);if e!=nil{return nil,"",e};headers,e:=s.headers();if e!=nil{return nil,"",e};data,h,e:=request(ctx,s.Client,"GET",endpoint,headers,nil);var up *UpstreamError;if errors.As(e,&up)&&up.Status==404{return nil,"",nil};if e!=nil{return nil,"",e};return ptr(string(data)),h.Get("ETag"),nil}
func(s *SpaceConnection)Write(ctx context.Context,path,text,etag string)error{endpoint,e:=s.path(path);if e!=nil{return e};headers,e:=s.headers();if e!=nil{return e};headers["Content-Type"]="text/markdown; charset=utf-8";if etag!=""{headers["If-Match"]=etag}else{headers["If-None-Match"]="*"};_,_,e=request(ctx,s.Client,"PUT",endpoint,headers,[]byte(text));return e}
func(s *SpaceConnection)Files(ctx context.Context)([]string,error){headers,e:=s.headers();if e!=nil{return nil,e};data,_,e:=request(ctx,s.Client,"GET",s.URL+"/.fs",headers,nil);if e!=nil{return nil,e};var value any;if e=json.Unmarshal(data,&value);e!=nil{return nil,e};out:=[]string{};switch v:=value.(type){case map[string]any:for k:=range v{out=append(out,k)};case []any:for _,p:=range v{if s,ok:=p.(string);ok{out=append(out,s)}else{out=append(out,stringValue(object(p)["name"]))}};default:return nil,errors.New("Invalid file list")};return out,nil}
type TaskConnector interface{VK(context.Context,string,string,any)(map[string]any,error);Paginated(context.Context,string,url.Values)([]map[string]any,error)}
type TaskConnection struct{URL,TokenFile string;Client *http.Client}
func NewTaskConnection()(*TaskConnection,error){config,e:=Configuration();if e!=nil{return nil,e};cfg:=object(config["vikunja"]);endpoint,e:=ServerURL(stringValue(cfg["url"]));if e!=nil{return nil,e};path:=stringValue(cfg["token_file"]);if _,e=credential(path);e!=nil{return nil,e};return &TaskConnection{endpoint,path,NewHTTPClient(30*time.Second)},nil}
func(t *TaskConnection)call(ctx context.Context,method,path string,body any)([]byte,http.Header,error){token,e:=credential(t.TokenFile);if e!=nil{return nil,nil,e};var data []byte;if body!=nil{data,e=json.Marshal(body);if e!=nil{return nil,nil,e}};return request(ctx,t.Client,method,t.URL+path,map[string]string{"Authorization":"Bearer "+token,"Content-Type":"application/json"},data)}
func(t *TaskConnection)VK(ctx context.Context,method,path string,body any)(map[string]any,error){data,_,e:=t.call(ctx,method,path,body);if e!=nil{return nil,e};out:=map[string]any{};if len(data)>0{e=json.Unmarshal(data,&out)};return out,e}
func(t *TaskConnection)Paginated(ctx context.Context,path string,params url.Values)([]map[string]any,error){out:=[]map[string]any{};if params==nil{params=url.Values{}};for page:=1;;page++{params.Set("page",strconv.Itoa(page));params.Set("per_page","100");data,h,e:=t.call(ctx,"GET",path+"?"+params.Encode(),nil);if e!=nil{return nil,e};var batch []map[string]any;if e=json.Unmarshal(data,&batch);e!=nil{return nil,e};out=append(out,batch...);pages:=page;if v:=h.Get("x-pagination-total-pages");v!=""{pages,e=strconv.Atoi(v);if e!=nil{return nil,e}};if page>=pages{return out,nil}}}
func number(v any)int64{switch n:=v.(type){case float64:return int64(n);case int:return int64(n);case int64:return n;case json.Number:i,_:=n.Int64();return i};return 0}
