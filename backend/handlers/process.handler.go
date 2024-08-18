package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"syscall"
	"time"

	"binalyze-test/configs"

	"binalyze-test/setup"
	"binalyze-test/types"

	"github.com/labstack/echo/v4"
	"golang.org/x/net/websocket"
)

type ProcessHandler struct {
	services *setup.ServiceDependencies
}

func UseAuthRoutes(routes *echo.Group, services *setup.ServiceDependencies) {
	p := ProcessHandler{services: services}
	routes.POST("/processes", p.getProcesses)
}

func getUserCount() (int, error) {
	var count int

	rows, err := configs.Db.Query("SELECT COUNT(*) FROM processes GROUP BY user")
	if err != nil {
		return count, err
	}
	defer rows.Close()

	for rows.Next() {
		if err := rows.Scan(&count); err != nil {
			return count, err
		}
	}

	return count, nil
}

func insertManyProcessQuery(processes []Process) error {
	sqlStr := "INSERT INTO processes (user, pid , cpuUsage , memoryUsage , residentMemorySize,  virtualMemorySize , state , totalTime , cpuTime , command , priority, createdAt , updatedAt) VALUES "

	upsertStatement := " ON CONFLICT (pid) DO UPDATE SET cpuUsage=excluded.cpuUsage , memoryUsage=excluded.memoryUsage , residentMemorySize=excluded.residentMemorySize, virtualMemorySize=excluded.virtualMemorySize , state=excluded.state , totalTime=excluded.totalTime,  cpuTime=excluded.cpuTime,  priority=excluded.priority, updatedAt=excluded.updatedAt ;"
	vals := []interface{}{}

	for _, process := range processes {

		sqlStr += "(?, ? , ? , ? ,?, ? , ? , ? , ? , ? , ? , ? , ?),"
		vals = append(vals, process.User, process.PID, process.CpuUsage, process.MemoryUsage, process.ResidentMemorySize, process.VirtualMemorySize, process.State, process.TotalTime, process.CpuTime, process.Command, process.Priority, time.Now(), time.Now())
	}

	// trim the last ,
	sqlStr = sqlStr[0 : len(sqlStr)-1]

	// concatenate upsert statement
	sqlStr += upsertStatement

	// format all vals at once
	_, err := configs.Db.Exec(sqlStr, vals...)
	if err != nil {
		return err
	}

	log.Println("Processes insertion was successful")

	return nil
}

func selectProcessesQuery(query string) ([]Process, error) {
	processes := []Process{}

	rows, err := configs.Db.Query(query)
	if err != nil {
		return processes, err
	}

	for rows.Next() {
		i := Process{}
		err = rows.Scan(&i.ID, &i.User, &i.PID, &i.CpuUsage, &i.MemoryUsage, &i.VirtualMemorySize, &i.ResidentMemorySize, &i.State, &i.TotalTime, &i.CpuTime, &i.Command, &i.Priority, &i.CreatedAt, &i.UpdatedAt)
		if err != nil {
			return processes, err
		}
		processes = append(processes, i)
	}

	// defer rows.Close()

	return processes, nil
}

func buildQuery(params url.Values) (string, int, int) {
	query := "SELECT * FROM processes"
	whereQuery := ""
	// build query based on params
	page, limit := 1, 10

	if params.Has("state") {
		whereQuery = fmt.Sprintf("%s state='%s' AND", whereQuery, params.Get("state"))
	}

	if params.Has("user") {
		whereQuery = fmt.Sprintf("%s user = '%s' AND", whereQuery, params.Get("user"))
	}

	if params.Has("search") {
		whereQuery = fmt.Sprintf("%s (user LIKE '%%%s%%' OR command LIKE '%%%s%%') AND", whereQuery, params.Get("search"), params.Get("search"))
	}

	if len(whereQuery) > 0 {
		// find last AND and remove
		lastAndIndex := strings.LastIndex(whereQuery, "AND")

		whereQuery = whereQuery[:lastAndIndex-1]

		whereQuery = "WHERE" + whereQuery + " "
	}

	if val, err := strconv.Atoi(params.Get("page")); err == nil {
		page = val
	}

	if val, err := strconv.Atoi(params.Get("limit")); err == nil {
		limit = val
	}

	offset := (page - 1) * limit

	query = fmt.Sprintf("%s %sLIMIT %d OFFSET %d", query, whereQuery, limit, offset)

	return query, limit, page
}

func getProcessReport() ([]ProcessUserReport, error) {
	data := []ProcessUserReport{}

	rows, err := configs.Db.Query("SELECT user, ROUND(SUM(cpuUsage),2) AS totalCpuUsage, ROUND(SUM(memoryUsage),2) AS totalMemoryUsage , COUNT(pid) as totalProcesses FROM processes GROUP BY user ORDER BY COUNT(pid) DESC")
	if err != nil {
		log.Println("Error getting users:", err)
		return data, err
	}
	defer rows.Close()

	for rows.Next() {
		report := ProcessUserReport{}
		err = rows.Scan(&report.User, &report.TotalCpuUsage, &report.TotalMemoryUsage, &report.TotalProcesses)
		if err != nil {
			log.Println("Error scanning users report usage:", err)
			return data, err
		}

		data = append(data, report)

	}

	return data, nil
}

func FetchAndInsertProcess() {
	processes := processHandler.GetProcesses()

	err := insertManyProcessQuery(processes)
	if err != nil {
		// log errror
		log.Println("err", err)
	}

	log.Println("Running processes fetched and inserted into db")
}

func (p ProcessHandler) getProcesses(c echo.Context) error {
	ctx := c.Request().Context()

	page, limit := 1, 10

	if val, err := strconv.Atoi(c.QueryParam("page")); err == nil {
		page = val
	}

	if val, err := strconv.Atoi(c.QueryParam("limit")); err == nil {
		limit = val
	}

	offset := (page - 1) * limit

	processList, err := p.services.ProcessService.GetProcesses(ctx, types.ProcessFilter{
		State:  c.QueryParam("state"),
		User:   c.QueryParam("user"),
		Search: c.QueryParam("search"),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return c.JSON(http.StatusOK, types.Response[*types.ProcessList]{
			Data:    nil,
			Success: false,
			Message: "Operation failed",
		})
	}

	processList.Limit = limit
	processList.Page = page

	return c.JSON(http.StatusOK, types.Response[*types.ProcessList]{
		Data:    processList,
		Success: true,
		Message: "Operation Successful",
	})
}

func GetProcessUsers(c echo.Context) error {
	users := []string{}

	rows, err := configs.Db.Query("SELECT user FROM processes GROUP BY user")
	if err != nil {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"data":    users,
			"success": false,
			"message": "Operation not successful",
		})
	}
	defer rows.Close()

	for rows.Next() {
		user := ""
		err = rows.Scan(&user)
		if err != nil {
			log.Println("Error getting scanning users:", err)
			return c.JSON(http.StatusOK, map[string]interface{}{
				"data":    users,
				"success": false,
				"message": "Operation not successful",
			})
		}
		users = append(users, user)

	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data":    users,
		"success": true,
		"message": "Operation successful",
	})
}

func GetProcessCounts(c echo.Context) error {
	data := map[string]int{
		"processCount": 0,
		"usersCount":   0,
	}

	processCount, err := getProcessCount()
	if err != nil {
		log.Println("Error getting processes:", err)
		return c.JSON(http.StatusOK, map[string]interface{}{
			"data":    data,
			"success": false,
			"message": "Operation not successful",
		})
	}

	data["processCount"] = processCount

	usersCount, err := getUserCount()
	if err != nil {
		log.Println("Error getting users:", err)
		return c.JSON(http.StatusOK, map[string]interface{}{
			"data":    data,
			"success": false,
			"message": "Operation not successful",
		})
	}

	data["usersCount"] = usersCount

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data":    data,
		"success": true,
		"message": "Operation successful",
	})
}

func GetProcessReports(c echo.Context) error {
	data, err := getProcessReport()
	if err != nil {
		log.Println("Error getting reports:", err)
		return c.JSON(http.StatusOK, map[string]interface{}{
			"data":    data,
			"success": false,
			"message": "Operation not successful",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data":    data,
		"success": true,
		"message": "Operation successful",
	})
}

func GetProcessRealTime(c echo.Context) error {
	websocket.Handler(func(ws *websocket.Conn) {
		defer ws.Close()
		for {
			// Write

			data := map[string]interface{}{
				"processes": []Process{},
				"reports":   []ProcessUserReport{},
			}

			query, _, _ := buildQuery(c.QueryParams())

			processes, err := selectProcessesQuery(query)
			if err == nil {
				data["processes"] = processes
			}

			reports, perr := getProcessReport()

			if perr == nil {
				data["reports"] = reports
			}

			var _json []byte

			_json, err = json.Marshal(data)
			if err != nil {
				c.Logger().Error(err)
			}

			msg := string(_json)

			time.Sleep(1 * time.Minute)

			err = websocket.Message.Send(ws, msg)
			if err != nil {
				if errors.Is(err, syscall.EPIPE) {
					break
				}
			}
		}
	}).ServeHTTP(c.Response(), c.Request())

	return nil
}
