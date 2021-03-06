package untis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ClientName is the name of this client communicating with the api
const ClientName = "Refundable"

// URL is the path this api is available at
const URL = "https://neilo.webuntis.com/WebUntis/jsonrpc.do?school=tgm"

// activeClients is a map that maps a user (the username) to the active client during an active session
var activeClients map[string]Client

// Client is the struct representing the client
type Client struct {
	// Username of the account the client uses
	Username string
	// Password of the account the client uses
	Password string
	// SessionID of the session the client is currently in
	SessionID string
	// PersonType of the account the client uses
	PersonType int
	// PersonID of the account the client uses
	PersonID int
	// Closed whether the current session is closed or not
	Closed bool
	// Authenticated whether the current session is active authenticated
	Authenticated bool
}

// Lesson represents a lesson out of a timetable
type Lesson struct {
	// Start is the start time of the lesson
	Start time.Time
	// End is the end time of the lesson
	End time.Time
	// ClassIDs are the ids of the classes participating
	ClassIDs []int
	// Classes are the names of all classes participating
	Classes []string
	// TeacherIDs are the ids of the teachers teaching
	TeacherIDs []int
	// Teachers are the names of all teachers teaching
	Teachers []string
	// RoomIDs are the room ids this lesson takes place in
	RoomIDs []int
	// Rooms are the room names this lesson takes place in
	Rooms []string
}

// CreateClient creates a new client to communicate with the API
// the username and password are used to authenticate the client at the service
func CreateClient(username, password string) *Client {
	client := Client{
		Username:      username,
		Password:      password,
		SessionID:     "",
		PersonType:    -1,
		PersonID:      -1,
		Closed:        false,
		Authenticated: false,
	}
	if activeClients == nil {
		activeClients = make(map[string]Client)
	}
	activeClients[username] = client
	return &client
}

// GetClient returns an active client using the corresponding username
func GetClient(username string) *Client {
	client := activeClients[username]
	return &client
}

// Authenticate authenticates the client at the untis service
func (client *Client) Authenticate() error {
	if client.Authenticated {
		return fmt.Errorf("already authenticated")
	}
	id := rand.Intn(math.MaxInt64)
	body, _ := json.Marshal(map[string]interface{}{
		"id":     id,
		"method": "authenticate",
		"params": map[string]string{
			"user":     client.Username,
			"password": client.Password,
			"client":   ClientName,
		},
		"jsonrpc": "2.0",
	})
	resp, err := http.Post(URL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	r := struct {
		JSONRPC string                 `json:"jsonrpc"`
		ID      string                 `json:"id"`
		Result  map[string]interface{} `json:"result"`
	}{}
	err = json.Unmarshal(respBody, &r)
	if err != nil {
		return err
	}
	personType, _ := r.Result["personType"].(float64)
	personID, _ := r.Result["personId"].(float64)
	if r.ID == strconv.Itoa(id) {
		client.SessionID = r.Result["sessionId"].(string)
		client.PersonType = int(personType)
		client.PersonID = int(personID)
		client.Authenticated = true
		client.Closed = false
		return nil
	}
	return fmt.Errorf("IDs not matching")
}

// GetTimetableOfTeacher returns a list of lessons the teacher logged in with the client has in between start and end
func (client Client) GetTimetableOfTeacher(start, end time.Time) ([]Lesson, error) {
	if !client.Authenticated {
		return nil, fmt.Errorf("not authenticated")
	}
	smonth := strconv.Itoa(int(start.Month()))
	if len(smonth) == 1 {
		smonth = "0" + smonth
	}
	sday := strconv.Itoa(start.Day())
	if len(sday) == 1 {
		sday = "0" + sday
	}
	emonth := strconv.Itoa(int(end.Month()))
	if len(emonth) == 1 {
		emonth = "0" + emonth
	}
	eday := strconv.Itoa(end.Day())
	if len(eday) == 1 {
		eday = "0" + eday
	}
	params := map[string]interface{}{
		"id":        client.PersonID,
		"type":      client.PersonType,
		"startDate": strconv.Itoa(start.Year()) + smonth + sday,
		"endDate":   strconv.Itoa(end.Year()) + emonth + eday,
	}
	resp, id, err := client.sendRequest("getTimetable", params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	r := struct {
		JSONRPC string `json:"jsonrpc"`
		ID      string `json:"id"`
		Result  []struct {
			ID        int `json:"id"`
			Date      int `json:"date"`
			StartTime int `json:"startTime"`
			EndTime   int `json:"endTime"`
			Kl        []struct {
				ID int `json:"id"`
			} `json:"kl"`
			Te []struct {
				ID int `json:"id"`
			} `json:"te"`
			Su []struct {
				ID int `json:"id"`
			} `json:"su"`
			Ro []struct {
				ID int `json:"id"`
			} `json:"ro"`
		} `json:"result"`
	}{}
	err = json.Unmarshal(respBody, &r)
	if err != nil {
		return nil, err
	}
	rid, _ := strconv.Atoi(r.ID)
	if rid == id {
		lessons := make([]Lesson, 0)
		for _, l := range r.Result {
			date := strconv.Itoa(l.Date)
			year, _ := strconv.Atoi(date[0:4])
			month, _ := strconv.Atoi(date[4:6])
			day, _ := strconv.Atoi(date[6:8])
			startTime := strconv.Itoa(l.StartTime)
			startHour, _ := strconv.Atoi(startTime[0 : len(startTime)-2])
			startMinute, _ := strconv.Atoi(startTime[len(startTime)-2:])
			endTime := strconv.Itoa(l.EndTime)
			endHour, _ := strconv.Atoi(endTime[0 : len(endTime)-2])
			endMinute, _ := strconv.Atoi(endTime[len(endTime)-2:])
			classIDArr := make([]int, 0)
			for _, kls := range l.Kl {
				classIDArr = append(classIDArr, kls.ID)
			}
			classArr, err := client.ResolveClasses(classIDArr)
			if err != nil {
				return nil, err
			}
			teachIDArr := make([]int, 0)
			for _, tes := range l.Te {
				teachIDArr = append(teachIDArr, tes.ID)
			}
			teachArr, err := client.ResolveTeachers(teachIDArr)
			if err != nil {
				return nil, err
			}
			roomIDArr := make([]int, 0)
			for _, ros := range l.Ro {
				roomIDArr = append(roomIDArr, ros.ID)
			}
			roomArr, err := client.ResolveRooms(roomIDArr)
			if err != nil {
				return nil, err
			}
			lessons = append(lessons, Lesson{
				Start:      time.Date(year, time.Month(month), day, startHour, startMinute, 0, 0, time.UTC),
				End:        time.Date(year, time.Month(month), day, endHour, endMinute, 0, 0, time.UTC),
				ClassIDs:   classIDArr,
				Classes:    classArr,
				TeacherIDs: teachIDArr,
				Teachers:   teachArr,
				RoomIDs:    roomIDArr,
				Rooms:      roomArr,
			})
		}
		return lessons, nil
	}
	return nil, fmt.Errorf("ids not matching")
}

// GetTimetableOfClass returns a list of lessons a specified class has in between start and end
func (client Client) GetTimetableOfClass(start, end time.Time, class string) ([]Lesson, error) {
	if !client.Authenticated {
		return nil, fmt.Errorf("not authenticated")
	}
	smonth := strconv.Itoa(int(start.Month()))
	if len(smonth) == 1 {
		smonth = "0" + smonth
	}
	sday := strconv.Itoa(start.Day())
	if len(sday) == 1 {
		sday = "0" + sday
	}
	emonth := strconv.Itoa(int(end.Month()))
	if len(emonth) == 1 {
		emonth = "0" + emonth
	}
	eday := strconv.Itoa(end.Day())
	if len(eday) == 1 {
		eday = "0" + eday
	}
	classID, _ := client.ResolveClassID(class)
	params := map[string]interface{}{
		"id":        classID,
		"type":      1,
		"startDate": strconv.Itoa(start.Year()) + smonth + sday,
		"endDate":   strconv.Itoa(end.Year()) + emonth + eday,
	}
	resp, id, err := client.sendRequest("getTimetable", params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	r := struct {
		JSONRPC string `json:"jsonrpc"`
		ID      string `json:"id"`
		Result  []struct {
			ID        int `json:"id"`
			Date      int `json:"date"`
			StartTime int `json:"startTime"`
			EndTime   int `json:"endTime"`
			Kl        []struct {
				ID int `json:"id"`
			} `json:"kl"`
			Te []struct {
				ID int `json:"id"`
			} `json:"te"`
			Su []struct {
				ID int `json:"id"`
			} `json:"su"`
			Ro []struct {
				ID int `json:"id"`
			} `json:"ro"`
		} `json:"result"`
	}{}
	err = json.Unmarshal(respBody, &r)
	if err != nil {
		return nil, err
	}
	rid, _ := strconv.Atoi(r.ID)
	if rid == id {
		lessons := make([]Lesson, 0)
		for _, l := range r.Result {
			date := strconv.Itoa(l.Date)
			year, _ := strconv.Atoi(date[0:4])
			month, _ := strconv.Atoi(date[4:6])
			day, _ := strconv.Atoi(date[6:8])
			startTime := strconv.Itoa(l.StartTime)
			startHour, _ := strconv.Atoi(startTime[0 : len(startTime)-2])
			startMinute, _ := strconv.Atoi(startTime[len(startTime)-2:])
			endTime := strconv.Itoa(l.EndTime)
			endHour, _ := strconv.Atoi(endTime[0 : len(endTime)-2])
			endMinute, _ := strconv.Atoi(endTime[len(endTime)-2:])
			classIDArr := make([]int, 0)
			for _, kls := range l.Kl {
				classIDArr = append(classIDArr, kls.ID)
			}
			classArr, err := client.ResolveClasses(classIDArr)
			if err != nil {
				return nil, err
			}
			teachIDArr := make([]int, 0)
			for _, tes := range l.Te {
				teachIDArr = append(teachIDArr, tes.ID)
			}
			teachArr, err := client.ResolveTeachers(teachIDArr)
			if err != nil {
				return nil, err
			}
			roomIDArr := make([]int, 0)
			for _, ros := range l.Ro {
				roomIDArr = append(roomIDArr, ros.ID)
			}
			roomArr, err := client.ResolveRooms(roomIDArr)
			if err != nil {
				return nil, err
			}
			lessons = append(lessons, Lesson{
				Start:      time.Date(year, time.Month(month), day, startHour, startMinute, 0, 0, time.UTC),
				End:        time.Date(year, time.Month(month), day, endHour, endMinute, 0, 0, time.UTC),
				ClassIDs:   classIDArr,
				Classes:    classArr,
				TeacherIDs: teachIDArr,
				Teachers:   teachArr,
				RoomIDs:    roomIDArr,
				Rooms:      roomArr,
			})
		}
		return lessons, nil
	}
	return nil, fmt.Errorf("ids not matching")
}

// GetTimetableOfSpecificTeacher returns a list of lessons a specified teacher has in between start and end
func (client Client) GetTimetableOfSpecificTeacher(start, end time.Time, teacher string) ([]Lesson, error) {
	if !client.Authenticated {
		return nil, fmt.Errorf("not authenticated")
	}
	smonth := strconv.Itoa(int(start.Month()))
	if len(smonth) == 1 {
		smonth = "0" + smonth
	}
	sday := strconv.Itoa(start.Day())
	if len(sday) == 1 {
		sday = "0" + sday
	}
	emonth := strconv.Itoa(int(end.Month()))
	if len(emonth) == 1 {
		emonth = "0" + emonth
	}
	eday := strconv.Itoa(end.Day())
	if len(eday) == 1 {
		eday = "0" + eday
	}
	id, err := client.ResolveTeacherID(teacher)
	if err != nil {
		return nil, err
	}
	params := map[string]interface{}{
		"id":        id,
		"type":      2,
		"startDate": strconv.Itoa(start.Year()) + smonth + sday,
		"endDate":   strconv.Itoa(end.Year()) + emonth + eday,
	}
	resp, id, err := client.sendRequest("getTimetable", params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	r := struct {
		JSONRPC string `json:"jsonrpc"`
		ID      string `json:"id"`
		Result  []struct {
			ID        int `json:"id"`
			Date      int `json:"date"`
			StartTime int `json:"startTime"`
			EndTime   int `json:"endTime"`
			Kl        []struct {
				ID int `json:"id"`
			} `json:"kl"`
			Te []struct {
				ID int `json:"id"`
			} `json:"te"`
			Su []struct {
				ID int `json:"id"`
			} `json:"su"`
			Ro []struct {
				ID int `json:"id"`
			} `json:"ro"`
		} `json:"result"`
	}{}
	err = json.Unmarshal(respBody, &r)
	if err != nil {
		return nil, err
	}
	rid, _ := strconv.Atoi(r.ID)
	if rid == id {
		lessons := make([]Lesson, 0)
		for _, l := range r.Result {
			date := strconv.Itoa(l.Date)
			year, _ := strconv.Atoi(date[0:4])
			month, _ := strconv.Atoi(date[4:6])
			day, _ := strconv.Atoi(date[6:8])
			startTime := strconv.Itoa(l.StartTime)
			startHour, _ := strconv.Atoi(startTime[0 : len(startTime)-2])
			startMinute, _ := strconv.Atoi(startTime[len(startTime)-2:])
			endTime := strconv.Itoa(l.EndTime)
			endHour, _ := strconv.Atoi(endTime[0 : len(endTime)-2])
			endMinute, _ := strconv.Atoi(endTime[len(endTime)-2:])
			classIDArr := make([]int, 0)
			for _, kls := range l.Kl {
				classIDArr = append(classIDArr, kls.ID)
			}
			classArr, err := client.ResolveClasses(classIDArr)
			if err != nil {
				return nil, err
			}
			teachIDArr := make([]int, 0)
			for _, tes := range l.Te {
				teachIDArr = append(teachIDArr, tes.ID)
			}
			teachArr, err := client.ResolveTeachers(teachIDArr)
			if err != nil {
				return nil, err
			}
			roomIDArr := make([]int, 0)
			for _, ros := range l.Ro {
				roomIDArr = append(roomIDArr, ros.ID)
			}
			roomArr, err := client.ResolveRooms(roomIDArr)
			if err != nil {
				return nil, err
			}
			lessons = append(lessons, Lesson{
				Start:      time.Date(year, time.Month(month), day, startHour, startMinute, 0, 0, time.UTC),
				End:        time.Date(year, time.Month(month), day, endHour, endMinute, 0, 0, time.UTC),
				ClassIDs:   classIDArr,
				Classes:    classArr,
				TeacherIDs: teachIDArr,
				Teachers:   teachArr,
				RoomIDs:    roomIDArr,
				Rooms:      roomArr,
			})
		}
		return lessons, nil
	}
	return nil, fmt.Errorf("ids not matching")
}

// ResolveTeachers converts an array of teacher ids into an array of teacher names
func (client Client) ResolveTeachers(ids []int) ([]string, error) {
	if !client.Authenticated {
		return nil, fmt.Errorf("not authenticated")
	}
	resp, id, err := client.sendRequest("getTeachers", map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	r := struct {
		JSONRPC string `json:"jsonrpc"`
		ID      string `json:"id"`
		Result  []struct {
			ID        int    `json:"id"`
			Name      string `json:"name"`
			Forename  string `json:"foreName"`
			Longname  string `json:"longName"`
			ForeColor string `json:"foreColor"`
			BackColor string `json:"backColor"`
		} `json:"result"`
	}{}
	err = json.Unmarshal(respBody, &r)
	if err != nil {
		return nil, err
	}
	rid, _ := strconv.Atoi(r.ID)
	if id == rid {
		teacher := make([]string, 0)
		for _, id := range ids {
			for _, res := range r.Result {
				if id == res.ID {
					teacher = append(teacher, res.Name)
				}
			}
		}
		return teacher, nil
	}
	return nil, fmt.Errorf("ids not matching")
}

// ResolveTeacherID converts a teacher name to the corersponding teacher id
func (client Client) ResolveTeacherID(teacher string) (int, error) {
	if !client.Authenticated {
		return -1, fmt.Errorf("not authenticated")
	}
	resp, id, err := client.sendRequest("getTeachers", map[string]interface{}{})
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return -1, err
	}
	r := struct {
		JSONRPC string `json:"jsonrpc"`
		ID      string `json:"id"`
		Result  []struct {
			ID        int    `json:"id"`
			Name      string `json:"name"`
			ForeName  string `json:"foreName"`
			LongName  string `json:"longName"`
			ForeColor string `json:"foreColor"`
			BackColor string `json:"backColor"`
		} `json:"result"`
	}{}
	err = json.Unmarshal(respBody, &r)
	if err != nil {
		return -1, err
	}
	rid, _ := strconv.Atoi(r.ID)
	if rid == id {
		split := strings.Split(teacher, " ")
		forename := split[0]
		longname := strings.ToUpper(split[1])
		for _, res := range r.Result {
			surname := strings.Split(res.LongName, " ")[0]
			if forename == res.ForeName && longname == surname {
				return res.ID, nil
			}
		}
		return -1, fmt.Errorf("teacher not found")
	}
	return -1, fmt.Errorf("ids not matching")
}

// ResolveRooms converts an array of room ids into an array of room names
func (client Client) ResolveRooms(ids []int) ([]string, error) {
	if !client.Authenticated {
		return nil, fmt.Errorf("not authenticated")
	}
	resp, id, err := client.sendRequest("getRooms", map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	r := struct {
		JSONRPC string `json:"jsonrpc"`
		ID      string `json:"id"`
		Result  []struct {
			ID        int    `json:"id"`
			Name      string `json:"name"`
			Longname  string `json:"longName"`
			ForeColor string `json:"foreColor"`
			BackColor string `json:"backColor"`
		} `json:"result"`
	}{}
	err = json.Unmarshal(respBody, &r)
	if err != nil {
		return nil, err
	}
	rid, _ := strconv.Atoi(r.ID)
	if id == rid {
		rooms := make([]string, 0)
		for _, id := range ids {
			for _, res := range r.Result {
				if id == res.ID {
					rooms = append(rooms, res.Name)
				}
			}
		}
		return rooms, nil
	}
	return nil, fmt.Errorf("ids not matching")
}

// ResolveClasses converts an array of class ids into an array of class names
func (client Client) ResolveClasses(ids []int) ([]string, error) {
	if !client.Authenticated {
		return nil, fmt.Errorf("not authenticated")
	}
	resp, id, err := client.sendRequest("getKlassen", map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	r := struct {
		JSONRPC string `json:"jsonrpc"`
		ID      string `json:"id"`
		Result  []struct {
			ID        int    `json:"id"`
			Name      string `json:"name"`
			Longname  string `json:"longName"`
			ForeColor string `json:"foreColor"`
			BackColor string `json:"backColor"`
			Teacher1  int    `json:"teacher1"`
			Teacher2  int    `json:"teacher2"`
		} `json:"result"`
	}{}
	err = json.Unmarshal(respBody, &r)
	if err != nil {
		return nil, err
	}
	rid, _ := strconv.Atoi(r.ID)
	if rid == id {
		classes := make([]string, 0)
		for _, id := range ids {
			for _, res := range r.Result {
				if id == res.ID {
					classes = append(classes, res.Name)
				}
			}
		}
		return classes, nil
	}
	return nil, fmt.Errorf("ids not matching")
}

// ResolveClassID converts a class name to the corresponding class id
func (client Client) ResolveClassID(class string) (int, error) {
	if !client.Authenticated {
		return -1, fmt.Errorf("not authenticated")
	}
	resp, id, err := client.sendRequest("getKlassen", map[string]interface{}{})
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return -1, err
	}
	r := struct {
		JSONRPC string `json:"jsonrpc"`
		ID      string `json:"id"`
		Result  []struct {
			ID        int    `json:"id"`
			Name      string `json:"name"`
			Longname  string `json:"longName"`
			ForeColor string `json:"foreColor"`
			BackColor string `json:"backColor"`
			Teacher1  int    `json:"teacher1"`
			Teacher2  int    `json:"teacher2"`
		} `json:"result"`
	}{}
	err = json.Unmarshal(respBody, &r)
	if err != nil {
		return -1, err
	}
	rid, _ := strconv.Atoi(r.ID)
	if rid == id {
		for _, res := range r.Result {
			if class == res.Name {
				return res.ID, nil
			}
		}
		return -1, fmt.Errorf("class not found")
	}
	return -1, fmt.Errorf("ids not matching")
}

// Close closes an authenticated connection to the untis api
func (client *Client) Close() error {
	if !client.Authenticated {
		return fmt.Errorf("not authenticated")
	}
	client.Closed = true
	client.Authenticated = false
	_, _, err := client.sendRequest("logout", map[string]interface{}{})
	if err != nil {
		return err
	}
	return nil
}

// DeleteClient deletes the current client out of the map of active clients
func (client Client) DeleteClient() {
	delete(activeClients, client.Username)
}

// sendRequest helps this api to send requests to the untis api
func (client Client) sendRequest(method string, params map[string]interface{}) (*http.Response, int, error) {
	id := rand.Intn(math.MaxInt64)
	body, _ := json.Marshal(map[string]interface{}{
		"id":      id,
		"method":  method,
		"params":  params,
		"jsonrpc": "2.0",
	})
	req, err := http.NewRequest("POST", URL, bytes.NewBuffer(body))
	if err != nil {
		return nil, -1, err
	}
	reqClient := http.Client{}
	req.AddCookie(&http.Cookie{Name: "JSESSIONID", Value: client.SessionID})
	resp, err := reqClient.Do(req)
	return resp, id, err
}

// GetLessonNrByStart computes the lesson number by its start time
func GetLessonNrByStart(start time.Time) int {
	switch start.Hour() {
	case 8:
		if start.Minute() == 0 {
			return 1
		} else if start.Minute() == 50 {
			return 2
		}
	case 9:
		return 3
	case 10:
		return 4
	case 11:
		return 5
	case 12:
		return 6
	case 13:
		return 7
	case 14:
		return 8
	case 15:
		return 9
	case 16:
		return 10
	case 17:
		if start.Minute() == 0 {
			return 11
		} else if start.Minute() == 45 {
			return 12
		}
	case 18:
		return 13
	case 19:
		return 14
	case 20:
		return 15
	}
	return -1
}

// GetLessonNrByEnd computes the lesson number by its end time
func GetLessonNrByEnd(end time.Time) int {
	switch end.Hour() {
	case 8:
		return 1
	case 9:
		return 2
	case 10:
		return 3
	case 11:
		return 4
	case 12:
		return 5
	case 13:
		return 6
	case 14:
		return 7
	case 15:
		return 8
	case 16:
		if end.Minute() == 0 {
			return 9
		} else if end.Minute() == 50 {
			return 10
		}
	case 17:
		return 11
	case 18:
		return 12
	case 19:
		return 13
	case 20:
		return 14
	case 21:
		return 15
	}
	return -1
}
