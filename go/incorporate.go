package paperwork

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strings"
)

var blockMarker = regexp.MustCompile(`<!-- paperwork-block:[^>]+ -->`)

func Incorporate(ctx context.Context, p Proposal, identity string, space Space, aliases, taskIDs map[string]string) ([]map[string]any, error) {
	notes, e := Render(p, identity, aliases, taskIDs)
	if e != nil {
		return nil, e
	}
	documentPath := "Paperwork/Documents/" + identity + ".md"
	document, _, e := space.Read(ctx, documentPath)
	if e != nil {
		return nil, e
	}
	if document == nil {
		link := "/.proxy/127.0.0.1:8089/v1/paperwork/documents/" + identity + "/original"
		if e = space.Write(ctx, documentPath, "# Documento original\n\n<a href=\""+link+"\" target=\"_blank\" rel=\"noopener\">Abrir original preservado</a>\n", ""); e != nil {
			return nil, e
		}
	}
	out := []map[string]any{}
	seen := map[string]bool{}
	order := []string{}
	for _, b := range p.Blocks {
		path := str(b.Context)
		if strings.HasPrefix(path, "Page/") {
			path = "Paperwork/Pages/" + path[5:]
		}
		if !seen[path] {
			order = append(order, path)
			seen[path] = true
		}
	}
	for _, page := range order {
		content := notes[page]
		path := page + ".md"
		previous, etag, e := space.Read(ctx, path)
		if e != nil {
			return nil, e
		}
		existing := str(previous)
		missing := []string{}
		for _, line := range splitlines(content) {
			if marker := blockMarker.FindString(line); marker != "" && strings.Contains(existing, marker) {
				continue
			}
			missing = append(missing, line)
		}
		if len(missing) == 0 {
			out = append(out, map[string]any{"path": page, "changed": false})
			continue
		}
		if previous != nil && etag == "" {
			return nil, errors.New("Server did not provide a revision; refusing unsafe overwrite")
		}
		updated := strings.TrimRight(existing, " \t\n\r\v\f") + "\n\n## Caderno\n\n" + strings.Join(missing, "\n") + "\n"
		if e = space.Write(ctx, path, updated, etag); e != nil {
			return nil, e
		}
		out = append(out, map[string]any{"path": page, "changed": true})
	}
	return out, nil
}
func (s *Store) IncorporateReviewed(ctx context.Context, identity, run string, allowDuplicates bool, space Space) (map[string]any, error) {
	p, raw, e := s.LoadProposal(identity)
	if e != nil {
		return nil, e
	}
	if e = s.CheckRevision(identity, run); e != nil {
		return nil, e
	}
	if len(p.PossibleDuplicates) > 0 && !allowDuplicates {
		return nil, errors.New("Review possible duplicate segments before incorporation")
	}
	hash := Hash([]byte(raw))
	var reserved, phase string
	e = s.DB.QueryRow("SELECT proposal_hash,phase FROM incorporation WHERE document_id=?", identity).Scan(&reserved, &phase)
	if e != nil && e != sql.ErrNoRows {
		return nil, e
	}
	if e == nil {
		if reserved != hash {
			return nil, errors.New("Document already incorporated with another revision; reconcile existing blocks before changing the tree")
		}
		if phase == "done" {
			return map[string]any{"notes": []any{}, "already_incorporated": true}, nil
		}
	}
	aliases, e := s.Aliases()
	if e != nil {
		return nil, e
	}
	if _, e = s.DB.Exec("INSERT OR IGNORE INTO incorporation VALUES (?,?,'pending')", identity, hash); e != nil {
		return nil, e
	}
	ids, e := s.EnsureLocal(p)
	if e != nil {
		return nil, e
	}
	notes, e := Incorporate(ctx, p, identity, space, aliases, ids)
	if e != nil {
		return nil, e
	}
	if _, e = s.DB.Exec("UPDATE incorporation SET phase='done' WHERE document_id=?", identity); e != nil {
		return nil, e
	}
	if e = s.RegisterSegments(identity, p); e != nil {
		return nil, e
	}
	return map[string]any{"notes": notes}, nil
}
