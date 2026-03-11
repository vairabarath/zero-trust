package admin

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"controller/state"
	"github.com/google/uuid"
)

func (s *Server) handleUIUsers(w http.ResponseWriter, r *http.Request) {
	db, ok := s.uiDB(w)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		rows, err := db.Query(`SELECT id, name, email, status, certificate_identity,
			CAST(created_at AS TEXT) as created_at
			FROM users ORDER BY name ASC`)
		if err != nil {
			http.Error(w, "failed to list users", http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		groupStmt, err := db.Prepare(state.Rebind(`SELECT group_id FROM user_group_members WHERE user_id = ?`))
		if err != nil {
			http.Error(w, "failed to list user groups", http.StatusInternalServerError)
			return
		}
		defer groupStmt.Close()
		out := []uiUser{}
		for rows.Next() {
			var id, name, email, status string
			var certID sql.NullString
			var created sql.NullString
			if err := rows.Scan(&id, &name, &email, &status, &certID, &created); err != nil {
				http.Error(w, "failed to read users", http.StatusInternalServerError)
				return
			}
			groupRows, _ := groupStmt.Query(id)
			groups := []string{}
			for groupRows.Next() {
				var gid string
				if err := groupRows.Scan(&gid); err == nil {
					groups = append(groups, gid)
				}
			}
			groupRows.Close()
			createdAt := ""
			if created.Valid {
				createdAt = created.String
			}
			out = append(out, uiUser{
				ID:                  id,
				Name:                name,
				Type:                "USER",
				DisplayLabel:        fmt.Sprintf("User: %s", name),
				Email:               email,
				Status:              strings.ToLower(status),
				Groups:              groups,
				CertificateIdentity: certID.String,
				CreatedAt:           createdAt,
			})
		}
		writeJSON(w, http.StatusOK, out)
	case http.MethodPost:
		var req struct {
			Name   string `json:"name"`
			Email  string `json:"email"`
			Status string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if req.Name == "" || req.Email == "" {
			http.Error(w, "name and email are required", http.StatusBadRequest)
			return
		}
		status := "active"
		if strings.ToLower(req.Status) == "inactive" {
			status = "inactive"
		}
		id := fmt.Sprintf("usr_%d", time.Now().UTC().UnixMilli())
		certID := "identity-" + uuid.NewString()
		createdAt := time.Now().UTC().Unix()
		if _, err := db.Exec(state.Rebind(`INSERT INTO users (id, name, email, certificate_identity, status, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`),
			id, req.Name, strings.ToLower(strings.TrimSpace(req.Email)), certID, status, "Member", createdAt, createdAt); err != nil {
			http.Error(w, "failed to create user", http.StatusBadRequest)
			return
		}
		if s.ACLNotify != nil {
			s.ACLNotify.NotifyPolicyChange()
		}
		writeJSON(w, http.StatusOK, uiUser{
			ID:                  id,
			Name:                req.Name,
			Type:                "USER",
			DisplayLabel:        fmt.Sprintf("User: %s", req.Name),
			Email:               req.Email,
			Status:              status,
			Groups:              []string{},
			CertificateIdentity: certID,
			CreatedAt:           dateStringFromUnix(createdAt),
		})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleUISubjects(w http.ResponseWriter, r *http.Request) {
	db, ok := s.uiDB(w)
	if !ok {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	subjectType := strings.ToUpper(r.URL.Query().Get("type"))
	subjects := []uiSubject{}
	if subjectType == "" || subjectType == "USER" {
		rows, _ := db.Query(`SELECT id, name FROM users ORDER BY name ASC`)
		if rows != nil {
			for rows.Next() {
				var id, name string
				if err := rows.Scan(&id, &name); err == nil {
					subjects = append(subjects, uiSubject{ID: id, Name: name, Type: "USER", DisplayLabel: fmt.Sprintf("User: %s", name)})
				}
			}
			rows.Close()
		}
	}
	if subjectType == "" || subjectType == "GROUP" {
		rows, _ := db.Query(`SELECT id, name FROM user_groups ORDER BY name ASC`)
		if rows != nil {
			for rows.Next() {
				var id, name string
				if err := rows.Scan(&id, &name); err == nil {
					subjects = append(subjects, uiSubject{ID: id, Name: name, Type: "GROUP", DisplayLabel: fmt.Sprintf("Group: %s", name)})
				}
			}
			rows.Close()
		}
	}
	if subjectType == "" || subjectType == "SERVICE" {
		rows, _ := db.Query(`SELECT id, name FROM service_accounts ORDER BY name ASC`)
		if rows != nil {
			for rows.Next() {
				var id, name string
				if err := rows.Scan(&id, &name); err == nil {
					subjects = append(subjects, uiSubject{ID: id, Name: name, Type: "SERVICE", DisplayLabel: fmt.Sprintf("Service: %s", name)})
				}
			}
			rows.Close()
		}
	}
	writeJSON(w, http.StatusOK, subjects)
}

func (s *Server) handleUIServiceAccounts(w http.ResponseWriter, r *http.Request) {
	db, ok := s.uiDB(w)
	if !ok {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	rows, err := db.Query(`SELECT id, name, status, associated_resource_count,
		CAST(created_at AS TEXT) as created_at
		FROM service_accounts ORDER BY name ASC`)
	if err != nil {
		http.Error(w, "failed to list service accounts", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	out := []uiServiceAccount{}
	for rows.Next() {
		var sa uiServiceAccount
		var created sql.NullString
		if err := rows.Scan(&sa.ID, &sa.Name, &sa.Status, &sa.AssociatedResourceCount, &created); err == nil {
			sa.Type = "SERVICE"
			sa.DisplayLabel = fmt.Sprintf("Service: %s", sa.Name)
			if created.Valid {
				sa.CreatedAt = created.String
			}
			out = append(out, sa)
		}
	}
	writeJSON(w, http.StatusOK, out)
}
