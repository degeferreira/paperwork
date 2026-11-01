package paperwork

import (
 "bytes"
 "crypto/sha256"
 "database/sql"
 "encoding/hex"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "os"
 "path/filepath"
 "sort"
 "strings"
 "time"

 "github.com/google/uuid"
 "golang.org/x/text/unicode/norm"
 _ "modernc.org/sqlite"
)

type Store struct { Root string; Originals string; DB *sql.DB }
func Open(root string)(*Store,error){
 if e:=os.MkdirAll(filepath.Join(root,"originals"),0700);e!=nil{return nil,e}
 db,e:=sql.Open("sqlite",filepath.Join(root,"paperwork.sqlite"));if e!=nil{return nil,e};db.SetMaxOpenConns(1)
 fail:=func(e error)(*Store,error){db.Close();return nil,e};var version int
 if e=db.QueryRow("PRAGMA user_version").Scan(&version);e!=nil{return fail(e)};if version>1{return fail(errors.New("Library schema newer than this Paperwork version"))}
 _,e=db.Exec(`PRAGMA busy_timeout=10000;
 CREATE TABLE IF NOT EXISTS documents(id TEXT PRIMARY KEY,name TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS segment_signatures(signature TEXT NOT NULL,document_id TEXT NOT NULL,context TEXT NOT NULL,PRIMARY KEY(signature,document_id,context));
 CREATE TABLE IF NOT EXISTS library_meta(key TEXT PRIMARY KEY,value TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS task_data(block_id TEXT PRIMARY KEY,title TEXT NOT NULL,project_id INTEGER NOT NULL,parent_block_id TEXT,done INTEGER NOT NULL DEFAULT 0);
 CREATE TABLE IF NOT EXISTS task_mentions(block_id TEXT PRIMARY KEY,task_id TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS document_metadata(document_id TEXT PRIMARY KEY,imported_at TEXT NOT NULL,captured_at TEXT,notebook TEXT,local_only INTEGER NOT NULL DEFAULT 1);
 CREATE TABLE IF NOT EXISTS proposals(document_id TEXT PRIMARY KEY,payload TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS incorporation(document_id TEXT PRIMARY KEY,proposal_hash TEXT NOT NULL,phase TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS tasks(block_id TEXT PRIMARY KEY,task_id TEXT UNIQUE NOT NULL,instance TEXT NOT NULL,remote_id INTEGER,phase TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS processing(id TEXT PRIMARY KEY,document_id TEXT NOT NULL,state TEXT NOT NULL,detail TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS runs(id TEXT PRIMARY KEY,document_id TEXT NOT NULL,created TEXT NOT NULL,pages TEXT NOT NULL,payload TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS corrections(block_id TEXT PRIMARY KEY,text TEXT NOT NULL);
 CREATE VIRTUAL TABLE IF NOT EXISTS search USING fts5(document_id UNINDEXED,text);
 CREATE TABLE IF NOT EXISTS reference_aliases(alias TEXT PRIMARY KEY,library_type TEXT NOT NULL,library_id INTEGER NOT NULL,item_key TEXT NOT NULL,note_path TEXT,retired INTEGER NOT NULL DEFAULT 0);
 PRAGMA user_version=1;`);if e!=nil{return fail(e)}
 if _,e=db.Exec("INSERT OR IGNORE INTO library_meta VALUES ('library_id',?),('task_sequence','0')",uuid.NewString());e!=nil{return fail(e)}
 return &Store{root,filepath.Join(root,"originals"),db},nil
}
func(s *Store)Close()error{return s.DB.Close()}
func Hash(data []byte)string{h:=sha256.Sum256(data);return hex.EncodeToString(h[:])}
func FileHash(path string)(string,error){f,e:=os.Open(path);if e!=nil{return "",e};defer f.Close();h:=sha256.New();if _,e=io.Copy(h,f);e!=nil{return "",e};return hex.EncodeToString(h.Sum(nil)),nil}
func now()string{return time.Now().UTC().Format("2006-01-02T15:04:05.000000+00:00")}
func(s *Store)Import(path string)(string,error){
 identity,e:=FileHash(path);if e!=nil{return "",e};target:=filepath.Join(s.Originals,identity)
 if _,e=os.Stat(target);os.IsNotExist(e){pending,e:=os.CreateTemp(s.Originals,".pending-");if e!=nil{return "",e};defer os.Remove(pending.Name());in,e:=os.Open(path);if e!=nil{pending.Close();return "",e};_,e=io.Copy(pending,in);in.Close();if e==nil{e=pending.Sync()};ce:=pending.Close();if e==nil{e=ce};if e!=nil{return "",e};digest,e:=FileHash(pending.Name());if e!=nil{return "",e};if digest!=identity{return "",errors.New("Original changed during import")};if e=os.Link(pending.Name(),target);e!=nil&&!os.IsExist(e){return "",e};dir,e:=os.Open(s.Originals);if e!=nil{return "",e};e=dir.Sync();dir.Close();if e!=nil{return "",e}}
 digest,e:=FileHash(target);if e!=nil{return "",e};if digest!=identity{return "",errors.New("Stored original failed integrity verification")}
 tx,e:=s.DB.Begin();if e!=nil{return "",e};defer tx.Rollback();if _,e=tx.Exec("INSERT OR IGNORE INTO documents VALUES (?,?)",identity,filepath.Base(path));e!=nil{return "",e};if _,e=tx.Exec("INSERT OR IGNORE INTO document_metadata(document_id,imported_at) VALUES (?,?)",identity,now());e!=nil{return "",e};return identity,tx.Commit()
}
// PythonJSON reproduces json.dumps(ensure_ascii=False, sort_keys=True) spacing
// for the duplicate signature contract. Historical proposal hashes never use it.
func PythonJSON(v any)[]byte {var b bytes.Buffer;enc:=json.NewEncoder(&b);enc.SetEscapeHTML(false);_ = enc.Encode(v);raw:=bytes.TrimSuffix(b.Bytes(),[]byte("\n"));var out bytes.Buffer;quoted,escape:=false,false;for _,c:=range raw{out.WriteByte(c);if quoted{if escape{escape=false}else if c=='\\'{escape=true}else if c=='"'{quoted=false}}else if c=='"'{quoted=true}else if c==','||c==':'{out.WriteByte(' ')}};return bytes.ReplaceAll(bytes.ReplaceAll(out.Bytes(),[]byte(`\u2028`),[]byte("\u2028")),[]byte(`\u2029`),[]byte("\u2029"))}
func signatures(p Proposal)map[string]string{groups:=map[string][]Block{};for _,b:=range p.Blocks{if b.Context!=nil{groups[*b.Context]=append(groups[*b.Context],b)}};result:=map[string]string{};for context,blocks:=range groups{positions:=map[string]int{};for i,b:=range blocks{positions[b.ID]=i};normalized:=[]any{};for _,b:=range blocks{var parent any;if n,ok:=positions[str(b.ParentID)];ok{parent=n};normalized=append(normalized,map[string]any{"kind":b.Kind,"text":strings.Join(strings.Fields(norm.NFKC.String(b.Text))," "),"parent":parent})};result[context]=Hash(PythonJSON([]any{context,normalized}))};return result}
func(s *Store)Propose(identity string,pages []string,context *string)(Proposal,error){
 var exists int;if e:=s.DB.QueryRow("SELECT 1 FROM documents WHERE id=?",identity).Scan(&exists);e!=nil{return Proposal{},e}
 p:=Parse(pages,identity,context);if context!=nil{p.AssignedContext=*context}
 for context,signature:=range signatures(p){rows,e:=s.DB.Query("SELECT s.document_id,d.name FROM segment_signatures s JOIN documents d ON s.document_id=d.id WHERE s.signature=? AND s.document_id!=?",signature,identity);if e!=nil{return p,e};for rows.Next(){var id,name string;if e=rows.Scan(&id,&name);e!=nil{rows.Close();return p,e};p.PossibleDuplicates=append(p.PossibleDuplicates,map[string]any{"context":context,"document_id":id,"name":name})};e=rows.Err();rows.Close();if e!=nil{return p,e}}
 payload,e:=json.Marshal(p);if e!=nil{return p,e};rawPages,e:=json.Marshal(pages);if e!=nil{return p,e};run:=uuid.NewString();tx,e:=s.DB.Begin();if e!=nil{return p,e};defer tx.Rollback()
 statements:=[]struct{q string;a []any}{{"INSERT INTO runs VALUES (?,?,?,?,?)",[]any{run,identity,now(),string(rawPages),string(payload)}},{"INSERT OR REPLACE INTO proposals VALUES (?,?)",[]any{identity,string(payload)}},{"DELETE FROM search WHERE document_id=?",[]any{identity}},{"INSERT INTO search VALUES (?,?)",[]any{identity,strings.Join(pages,"\n")}}}
 for _,st:=range statements{if _,e=tx.Exec(st.q,st.a...);e!=nil{return p,e}};if e=tx.Commit();e!=nil{return p,e};p.RunID=run;return p,nil
}
// LoadProposal returns the unmodified TEXT for hashing; decoding is separate.
func(s *Store)LoadProposal(identity string)(Proposal,string,error){var raw string;e:=s.DB.QueryRow("SELECT payload FROM proposals WHERE document_id=?",identity).Scan(&raw);if e!=nil{return Proposal{},"",e};var p Proposal;e=json.Unmarshal([]byte(raw),&p);return p,raw,e}
func(s *Store)CheckRevision(identity,run string)error{var latest string;if e:=s.DB.QueryRow("SELECT id FROM runs WHERE document_id=? ORDER BY rowid DESC LIMIT 1",identity).Scan(&latest);e!=nil||run==""||run!=latest{return errors.New("Load and review the current proposal; its revision changed or was not supplied")};return nil}
func(s *Store)RegisterSegments(identity string,p Proposal)error{tx,e:=s.DB.Begin();if e!=nil{return e};defer tx.Rollback();for context,signature:=range signatures(p){if _,e=tx.Exec("INSERT OR IGNORE INTO segment_signatures VALUES (?,?,?)",signature,identity,context);e!=nil{return e}};return tx.Commit()}
func queryMaps(db *sql.DB,q string,args ...any)([]map[string]any,error){rows,e:=db.Query(q,args...);if e!=nil{return nil,e};defer rows.Close();cols,e:=rows.Columns();if e!=nil{return nil,e};out:=[]map[string]any{};for rows.Next(){vals:=make([]any,len(cols));dest:=make([]any,len(cols));for i:=range vals{dest[i]=&vals[i]};if e=rows.Scan(dest...);e!=nil{return nil,e};m:=map[string]any{};for i,v:=range vals{if b,ok:=v.([]byte);ok{v=string(b)};m[cols[i]]=v};out=append(out,m)};return out,rows.Err()}
func(s *Store)Search(q string)([]map[string]any,error){return queryMaps(s.DB,"SELECT document_id AS document,snippet(search,1,'[',']','…',24) AS excerpt FROM search WHERE search MATCH ? LIMIT 50",`"`+strings.ReplaceAll(q,`"`,`""`)+`"`)}
func sortedKeys[V any](m map[string]V)[]string{keys:=make([]string,0,len(m));for k:=range m{keys=append(keys,k)};sort.Strings(keys);return keys}
func require(condition bool,message string)error{if !condition{return fmt.Errorf("%s",message)};return nil}
