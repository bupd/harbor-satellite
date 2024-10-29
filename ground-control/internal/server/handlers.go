package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"container-registry.com/harbor-satellite/ground-control/internal/database"
	"container-registry.com/harbor-satellite/ground-control/internal/models"
	"container-registry.com/harbor-satellite/ground-control/internal/utils"
	"container-registry.com/harbor-satellite/ground-control/reg/harbor"
	"github.com/gorilla/mux"
)

type RegisterSatelliteParams struct {
	Name   string    `json:"name"`
	Groups *[]string `json:"groups,omitempty"`
}
type SatelliteGroupParams struct {
	Satellite string `json:"satellite"`
	Group     string `json:"group"`
}

func DecodeRequestBody(r *http.Request, v interface{}) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return &AppError{
			Message: "Invalid request body",
			Code:    http.StatusBadRequest,
		}
	}
	return nil
}

func (s *Server) Ping(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("pong"))
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	err := s.db.Ping()
	if err != nil {
		log.Printf("error pinging db: %v", err)
		msg, _ := json.Marshal(map[string]string{"status": "unhealthy"})
		http.Error(w, string(msg), http.StatusBadRequest)
		return
	}

	WriteJSONResponse(w, http.StatusOK, map[string]string{"status": "healthy"})
}

func (s *Server) groupsSyncHandler(w http.ResponseWriter, r *http.Request) {
	var req models.StateArtifact
	if err := DecodeRequestBody(r, &req); err != nil {
		log.Println(err)
		HandleAppError(w, err)
		return
	}

	// Start a new transaction
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		log.Println(err)
		HandleAppError(w, err)
		return
	}
	// Create a new Queries object bound to the transaction
	q := s.dbQueries.WithTx(tx)
	// Ensure proper transaction handling with defer
	defer func() {
		if p := recover(); p != nil {
			// If there's a panic, rollback the transaction
			tx.Rollback()
		} else if err != nil {
			tx.Rollback() // Rollback transaction on error
		}
	}()
	projects := utils.GetProjectNames(&req.Artifacts)
	params := database.CreateGroupParams{
		GroupName:   req.Group,
		RegistryUrl: os.Getenv("HARBOR_URL"),
		Projects:    projects,
	}
	result, err := q.CreateGroup(r.Context(), params)
	if err != nil {
		log.Println(err)
		HandleAppError(w, err)
		return
	}
	satellites, err := q.GroupSatelliteList(r.Context(), result.ID)
	if err != nil {
		log.Println(err)
		HandleAppError(w, err)
		return
	}

	for _, satellite := range satellites {
		robotAcc, err := q.GetRobotAccBySatelliteID(r.Context(), satellite.SatelliteID)
		if err != nil {
			log.Println(err)
			HandleAppError(w, err)
			return
		}
		// update robot account projects permission
		_, err = utils.UpdateRobotProjects(r.Context(), projects, robotAcc.RobotName, robotAcc.RobotID)
		if err != nil {
			log.Println(err)
			HandleAppError(w, err)
			return
		}
	}

	// check if project satellite exists and if does not exist create project satellite
	satExist, err := harbor.GetProject(r.Context(), "satellite")
	if err != nil {
		log.Println(err)
		err := &AppError{
			Message: fmt.Sprintf("Error: Checking satellite project: %v", err),
			Code:    http.StatusBadGateway,
		}
		HandleAppError(w, err)
		tx.Rollback()
		return
	}
	if !satExist {
		_, err := harbor.CreateSatelliteProject(r.Context())
		if err != nil {
			log.Println(err)
			err := &AppError{
				Message: fmt.Sprintf("Error: creating satellite project: %v", err),
				Code:    http.StatusBadGateway,
			}
			HandleAppError(w, err)
			tx.Rollback()
			return
		}
	}

	// Create State Artifact for the group
	err = utils.CreateStateArtifact(&req)
	if err != nil {
		log.Println(err)
		HandleAppError(w, err)
		return
	}

	tx.Commit()
	WriteJSONResponse(w, http.StatusOK, result)
}

func (s *Server) registerSatelliteHandler(w http.ResponseWriter, r *http.Request) {
	var req RegisterSatelliteParams
	if err := DecodeRequestBody(r, &req); err != nil {
		log.Println(err)
		HandleAppError(w, err)
		return
	}

	if len(req.Name) < 1 {
		log.Println("name should be atleast one character long.")
		err := &AppError{
			Message: fmt.Sprintf("Error: name should be atleast one character long."),
			Code:    http.StatusBadRequest,
		}
		HandleAppError(w, err)
		return
	}

	// Start a new transaction
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		log.Println(err)
		HandleAppError(w, err)
		return
	}
	// Create a new Queries object bound to the transaction
	q := s.dbQueries.WithTx(tx)
	// Ensure proper transaction handling with defer
	defer func() {
		if p := recover(); p != nil {
			// If there's a panic, rollback the transaction
			tx.Rollback()
		} else if err != nil {
			tx.Rollback() // Rollback transaction on error
		}
	}()
	// Create satellite
	satellite, err := q.CreateSatellite(r.Context(), req.Name)
	if err != nil {
		log.Println(err)
		err := &AppError{
			Message: fmt.Sprintf("Error: %v", err.Error()),
			Code:    http.StatusBadRequest,
		}
		HandleAppError(w, err)
		tx.Rollback()
		return
	}

	// Check if Groups is nil before dereferencing
	if req.Groups != nil {
		// Add satellite to groups
		for _, groupName := range *req.Groups {
			// check if groups are declared in replication
			replications, err := harbor.ListReplication(r.Context(), harbor.ListParams{
				Q: fmt.Sprintf("name=%s", groupName),
			})
			if len(replications) < 1 {
				if err != nil {
					log.Println(err)
					err := &AppError{
						Message: fmt.Sprintf("Error: Group Name: %s, does not exist in replication, Please give a Valid Group Name", groupName),
						Code:    http.StatusBadRequest,
					}
					HandleAppError(w, err)
					tx.Rollback()
					return
				}
			}
			group, err := q.GetGroupByName(r.Context(), groupName)
			if err != nil {
				log.Println(err)
				err := &AppError{
					Message: fmt.Sprintf("Error: Invalid Group Name: %v", groupName),
					Code:    http.StatusBadRequest,
				}
				HandleAppError(w, err)
				tx.Rollback()
				return
			}
			err = q.AddSatelliteToGroup(r.Context(), database.AddSatelliteToGroupParams{
				SatelliteID: satellite.ID,
				GroupID:     group.ID,
			})
			if err != nil {
				log.Println(err)
				HandleAppError(w, err)
				tx.Rollback()
				return
			}
		}
	}

	// check if project satellite exists and if does not exist create project satellite
	satExist, err := harbor.GetProject(r.Context(), "satellite")
	if err != nil {
		log.Println(err)
		err := &AppError{
			Message: fmt.Sprintf("Error: Checking satellite project: %v", err),
			Code:    http.StatusBadGateway,
		}
		HandleAppError(w, err)
		tx.Rollback()
		return
	}
	if !satExist {
		_, err := harbor.CreateSatelliteProject(r.Context())
		if err != nil {
			log.Println(err)
			err := &AppError{
				Message: fmt.Sprintf("Error: creating satellite project: %v", err),
				Code:    http.StatusBadGateway,
			}
			HandleAppError(w, err)
			tx.Rollback()
			return
		}
	}

	// Create Robot Account for Satellite
	projects := []string{"satellite"}
	rbt, err := utils.CreateRobotAccForSatellite(r.Context(), projects, satellite.Name)
	if err != nil {
		log.Println(err)
		err := &AppError{
			Message: fmt.Sprintf("Error: creating robot account %v", err),
			Code:    http.StatusBadRequest,
		}
		HandleAppError(w, err)
		tx.Rollback()
		return
	}
	// Add Robot Account to database
	params := database.AddRobotAccountParams{
		RobotName:   rbt.Name,
		RobotSecret: rbt.Secret,
		RobotID:     strconv.Itoa(int(rbt.ID)),
		SatelliteID: satellite.ID,
	}
	_, err = q.AddRobotAccount(r.Context(), params)
	if err != nil {
		log.Println(err)
		err := &AppError{
			Message: fmt.Sprintf("Error: adding robot account to DB %v", err.Error()),
			Code:    http.StatusInternalServerError,
		}
		HandleAppError(w, err)
		tx.Rollback()
		return
	}

	// Add token to DB
	token, err := GenerateRandomToken(32)
	if err != nil {
		log.Println(err)
		HandleAppError(w, err)
		return
	}
	tk, err := q.AddToken(r.Context(), database.AddTokenParams{
		SatelliteID: satellite.ID,
		Token:       token,
	})
	if err != nil {
		log.Println("error in token")
		log.Println(err)
		HandleAppError(w, err)
		tx.Rollback()
		return
	}

	tx.Commit()
	WriteJSONResponse(w, http.StatusOK, tk)
}

func (s *Server) ztrHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	// Start a new transaction
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		log.Println(err)
		HandleAppError(w, err)
		return
	}

	q := s.dbQueries.WithTx(tx)

	defer func() {
		if p := recover(); p != nil {
			// If there's a panic, rollback the transaction
			tx.Rollback()
			panic(p) // Re-throw the panic after rolling back
		} else if err != nil {
			tx.Rollback() // Rollback transaction on error
		}
	}()

	satelliteID, err := q.GetSatelliteIDByToken(r.Context(), token)
	if err != nil {
		log.Println("Invalid Satellite Token")
		log.Println(err)
		err := &AppError{
			Message: "Error: Invalid Token",
			Code:    http.StatusBadRequest,
		}
		HandleAppError(w, err)
		tx.Rollback()
		return
	}

	err = q.DeleteToken(r.Context(), token)
	if err != nil {
		log.Println("error deleting token")
		log.Println(err)
		err := &AppError{
			Message: "Error: Error deleting token",
			Code:    http.StatusInternalServerError,
		}
		HandleAppError(w, err)
		tx.Rollback()
		return
	}

	robot, err := q.GetRobotAccBySatelliteID(r.Context(), satelliteID)
	if err != nil {
		log.Println("Robot Account Not Found")
		log.Println(err)
		err := &AppError{
			Message: "Error: Robot Account Not Found for Satellite",
			Code:    http.StatusInternalServerError,
		}
		HandleAppError(w, err)
		tx.Rollback()
		return
	}

	// groups attached to satellite
	groups, err := q.SatelliteGroupList(r.Context(), satelliteID)
	if err != nil {
		log.Printf("failed to list groups for satellite: %v, %v", satelliteID, err)
		log.Println(err)
		err := &AppError{
			Message: "Error: Satellite Groups List Failed",
			Code:    http.StatusInternalServerError,
		}
		HandleAppError(w, err)
		tx.Rollback()
		return
	}

	var states []string
	for _, group := range groups {
		grp, err := q.GetGroupByID(r.Context(), group.GroupID)
		if err != nil {
			log.Printf("failed to get group by ID: %v, %v", group.GroupID, err)
			log.Println(err)
			err := &AppError{
				Message: "Error: Get Group By ID Failed",
				Code:    http.StatusInternalServerError,
			}
			HandleAppError(w, err)
			tx.Rollback()
			return
		}
		state := utils.AssembleGroupState(grp.GroupName)
		states = append(states, state)
	}

	result := models.ZtrResult{
		States: states,
		Auth: models.Account{
			Name:     robot.RobotName,
			Secret:   robot.RobotSecret,
			Registry: os.Getenv("HARBOR_URL"),
		},
	}

	tx.Commit()
	WriteJSONResponse(w, http.StatusOK, result)
}

func (s *Server) listSatelliteHandler(w http.ResponseWriter, r *http.Request) {
	result, err := s.dbQueries.ListSatellites(r.Context())
	if err != nil {
		log.Printf("Error: Failed to List Satellites: %v", err)
		err := &AppError{
			Message: "Error: Failed to List Satellites",
			Code:    http.StatusInternalServerError,
		}
		HandleAppError(w, err)
		return
	}

	WriteJSONResponse(w, http.StatusOK, result)
}

func (s *Server) GetSatelliteByName(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	satellite := vars["satellite"]

	result, err := s.dbQueries.GetSatelliteByName(r.Context(), satellite)
	if err != nil {
		log.Printf("error: failed to get satellite: %v", err)
		err := &AppError{
			Message: "Error: Failed to Get Satellite",
			Code:    http.StatusInternalServerError,
		}
		HandleAppError(w, err)
		return
	}

	WriteJSONResponse(w, http.StatusOK, result)
}

func (s *Server) DeleteSatelliteByName(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	satellite := vars["satellite"]

	err := s.dbQueries.DeleteSatelliteByName(r.Context(), satellite)
	if err != nil {
		log.Printf("error: failed to delete satellite: %v", err)
		err := &AppError{
			Message: "Error: Failed to Delete Satellite",
			Code:    http.StatusInternalServerError,
		}
		HandleAppError(w, err)
		return
	}

	WriteJSONResponse(w, http.StatusOK, map[string]string{})
}

func (s *Server) addSatelliteToGroup(w http.ResponseWriter, r *http.Request) {
	var req SatelliteGroupParams
	if err := DecodeRequestBody(r, &req); err != nil {
		HandleAppError(w, err)
		return
	}

	sat, err := s.dbQueries.GetSatelliteByName(r.Context(), req.Satellite)
	if err != nil {
		log.Printf("Error: Satellite Not Found: %v", err)
		err := &AppError{
			Message: "Error: Satellite Not Found",
			Code:    http.StatusBadRequest,
		}
		HandleAppError(w, err)
		return
	}
	grp, err := s.dbQueries.GetGroupByName(r.Context(), req.Group)
	if err != nil {
		log.Printf("Error: Group Not Found: %v", err)
		err := &AppError{
			Message: "Error: Group Not Found",
			Code:    http.StatusBadRequest,
		}
		HandleAppError(w, err)
		return
	}

	params := database.AddSatelliteToGroupParams{
		SatelliteID: int32(sat.ID),
		GroupID:     int32(grp.ID),
	}

	err = s.dbQueries.AddSatelliteToGroup(r.Context(), params)
	if err != nil {
		log.Printf("Error: Failed to Add Satellite to Group: %v", err)
		err := &AppError{
			Message: "Error: Failed to Add Satellite to Group",
			Code:    http.StatusInternalServerError,
		}
		HandleAppError(w, err)
		return
	}

	WriteJSONResponse(w, http.StatusOK, map[string]string{})
}

func (s *Server) removeSatelliteFromGroup(w http.ResponseWriter, r *http.Request) {
	var req SatelliteGroupParams
	if err := DecodeRequestBody(r, &req); err != nil {
		HandleAppError(w, err)
		return
	}

	sat, err := s.dbQueries.GetSatelliteByName(r.Context(), req.Satellite)
	if err != nil {
		log.Printf("Error: Satellite Not Found: %v", err)
		err := &AppError{
			Message: "Error: Satellite Not Found",
			Code:    http.StatusBadRequest,
		}
		HandleAppError(w, err)
		return
	}
	grp, err := s.dbQueries.GetGroupByName(r.Context(), req.Group)
	if err != nil {
		log.Printf("Error: Group Not Found: %v", err)
		err := &AppError{
			Message: "Error: Group Not Found",
			Code:    http.StatusBadRequest,
		}
		HandleAppError(w, err)
		return
	}

	params := database.RemoveSatelliteFromGroupParams{
		SatelliteID: int32(sat.ID),
		GroupID:     int32(grp.ID),
	}

	err = s.dbQueries.RemoveSatelliteFromGroup(r.Context(), params)
	if err != nil {
		log.Printf("error: failed to remove satellite from group: %v", err)
		err := &AppError{
			Message: "Error: Failed to Remove Satellite from Group",
			Code:    http.StatusInternalServerError,
		}
		HandleAppError(w, err)
		return
	}

	WriteJSONResponse(w, http.StatusOK, map[string]string{})
}

func (s *Server) listGroupHandler(w http.ResponseWriter, r *http.Request) {
	result, err := s.dbQueries.ListGroups(r.Context())
	if err != nil {
		HandleAppError(w, err)
		return
	}

	WriteJSONResponse(w, http.StatusOK, result)
}

func (s *Server) getGroupHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	group := vars["group"]

	result, err := s.dbQueries.GetGroupByName(r.Context(), group)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	WriteJSONResponse(w, http.StatusOK, result)
}

// creates a unique random API token of the specified length in bytes.
func GenerateRandomToken(charLength int) (string, error) {
	// The number of bytes needed to generate a token with the required number of hex characters
	byteLength := charLength / 2

	// Create a byte slice of the required length
	token := make([]byte, byteLength)
	_, err := rand.Read(token)
	if err != nil {
		return "", err
	}

	// Return the token as a hex-encoded string
	return hex.EncodeToString(token), nil
}

func GetAuthToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		err := &AppError{
			Message: "Authorization header missing",
			Code:    http.StatusUnauthorized,
		}
		return "", err
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		err := &AppError{
			Message: "Invalid Authorization header format",
			Code:    http.StatusUnauthorized,
		}
		return "", err
	}
	token := parts[1]

	return token, nil
}
