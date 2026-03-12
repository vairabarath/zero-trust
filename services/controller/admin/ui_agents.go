package admin

import "net/http"

func (s *Server) handleUIAgents(w http.ResponseWriter, r *http.Request) {
	db, ok := s.uiDB(w)
	if !ok {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	rows, err := db.Query(`SELECT id, name, status, version, hostname, remote_network_id FROM tunnelers ORDER BY name ASC`)
	if err != nil {
		http.Error(w, "failed to list agents", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	out := []uiAgent{}
	for rows.Next() {
		var t uiAgent
		if err := rows.Scan(&t.ID, &t.Name, &t.Status, &t.Version, &t.Hostname, &t.RemoteNetworkID); err == nil {
			out = append(out, t)
		}
	}
	writeJSON(w, http.StatusOK, out)
}
