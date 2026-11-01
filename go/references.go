package paperwork

import (
 "database/sql"
 "errors"
 "regexp"
 "strings"
)
type Alias struct {Alias string `json:"alias"`;LibraryType string `json:"library_type"`;LibraryID int `json:"library_id"`;ItemKey string `json:"item_key"`;NotePath *string `json:"note_path"`}
var aliasRE=regexp.MustCompile(`^[\pL\pN_-]{1,100}$`)
var itemRE=regexp.MustCompile(`^[A-Z0-9]{8}$`)
func validateAlias(a Alias)error{if !aliasRE.MatchString(a.Alias)||(a.LibraryType!="users"&&a.LibraryType!="groups")||a.LibraryID<=0||!itemRE.MatchString(a.ItemKey){return errors.New("Invalid Zotero alias or identity")};if a.NotePath!=nil&&*a.NotePath!=""&&!safeNote(*a.NotePath){return errors.New("Invalid literature note path")};return nil}
func registerAlias(tx *sql.Tx,a Alias)error{if e:=validateAlias(a);e!=nil{return e};var typ,key string;var id,retired int;e:=tx.QueryRow("SELECT library_type,library_id,item_key,retired FROM reference_aliases WHERE alias=?",a.Alias).Scan(&typ,&id,&key,&retired);if e!=nil&&e!=sql.ErrNoRows{return e};if e==nil&&(typ!=a.LibraryType||id!=a.LibraryID||key!=a.ItemKey||retired!=0){return errors.New("Alias already assigned or retired; it cannot be reused")};_,e=tx.Exec("INSERT OR IGNORE INTO reference_aliases(alias,library_type,library_id,item_key,note_path) VALUES (?,?,?,?,?)",a.Alias,a.LibraryType,a.LibraryID,a.ItemKey,a.NotePath);return e}
func(s *Store)RegisterAliases(entries []Alias)error{tx,e:=s.DB.Begin();if e!=nil{return e};defer tx.Rollback();for _,a:=range entries{if e=registerAlias(tx,a);e!=nil{return e}};return tx.Commit()}
func(s *Store)Aliases()(map[string]string,error){rows,e:=s.DB.Query("SELECT alias,note_path FROM reference_aliases WHERE retired=0 AND note_path IS NOT NULL");if e!=nil{return nil,e};defer rows.Close();out:=map[string]string{};for rows.Next(){var alias,path string;if e=rows.Scan(&alias,&path);e!=nil{return nil,e};out[alias]=path};return out,rows.Err()}
func(s *Store)Resolve(p Proposal)([]map[string]any,error){out:=[]map[string]any{};for _,b:=range p.Blocks{for _,m:=range b.References{var typ,key string;var path sql.NullString;var id,retired int;e:=s.DB.QueryRow("SELECT library_type,library_id,item_key,note_path,retired FROM reference_aliases WHERE alias=?",m.Alias).Scan(&typ,&id,&key,&path,&retired);if e!=nil&&e!=sql.ErrNoRows{return nil,e};var identity,note any;if e==nil&&retired==0{identity=map[string]any{"library_type":typ,"library_id":id,"item_key":key};if path.Valid{note=path.String}};out=append(out,map[string]any{"block_id":b.ID,"alias":m.Alias,"page":m.Page,"identity":identity,"note_path":note})}};return out,nil}
func firstWord(s string)string{v:=strings.Fields(s);if len(v)>0{return v[0]};return ""}
