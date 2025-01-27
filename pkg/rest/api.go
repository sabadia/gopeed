package rest

import (
	"io"
	"net/http"
	"net/url"
	"runtime"
	"strconv"

	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/download"
	"github.com/GopeedLab/gopeed/pkg/rest/model"
	"github.com/gorilla/mux"
)

func Info(w http.ResponseWriter, r *http.Request) {
	info := map[string]any{
		"version":  base.Version,
		"runtime":  runtime.Version(),
		"os":       runtime.GOOS,
		"arch":     runtime.GOARCH,
		"inDocker": base.InDocker == "true",
	}

	statusCounts := map[base.Status]int{
		base.DownloadStatusReady:   0,
		base.DownloadStatusRunning: 0,
		base.DownloadStatusPause:   0,
		base.DownloadStatusWait:    0,
		base.DownloadStatusError:   0,
		base.DownloadStatusDone:    0,
	}

	tasks := Downloader.GetTasksByFilter(nil)
	for _, task := range tasks {
		statusCounts[task.Status]++
	}

	info["statusCounts"] = statusCounts
	info["totalTasks"] = len(tasks)

	WriteJson(w, model.NewOkResult(info))
}

func Resolve(w http.ResponseWriter, r *http.Request) {
	var req base.Request
	if ReadJson(r, w, &req) {
		rr, err := Downloader.Resolve(&req)
		if err != nil {
			WriteJson(w, model.NewErrorResult(err.Error()))
			return
		}
		WriteJson(w, model.NewOkResult(rr))
	}
}

func CreateTask(w http.ResponseWriter, r *http.Request) {
	var req model.CreateTask
	if ReadJson(r, w, &req) {
		var (
			taskId string
			err    error
		)
		if req.Rid != "" {
			taskId, err = Downloader.Create(req.Rid, req.Opt)
		} else if req.Req != nil {
			taskId, err = Downloader.CreateDirect(req.Req, req.Opt)
		} else {
			WriteJson(w, model.NewErrorResult("param invalid: rid or req", model.CodeInvalidParam))
			return
		}
		if err != nil {
			WriteJson(w, model.NewErrorResult(err.Error()))
			return
		}
		WriteJson(w, model.NewOkResult(taskId))
	}
}

func CreateTaskBatch(w http.ResponseWriter, r *http.Request) {
	var req model.CreateTaskBatch
	if ReadJson(r, w, &req) {
		if len(req.Reqs) == 0 {
			WriteJson(w, model.NewErrorResult("param invalid: reqs", model.CodeInvalidParam))
			return
		}
		taskIds, err := Downloader.CreateDirectBatch(req.Reqs, req.Opt)
		if err != nil {
			WriteJson(w, model.NewErrorResult(err.Error()))
			return
		}
		WriteJson(w, model.NewOkResult(taskIds))
	}
}

func PauseTask(w http.ResponseWriter, r *http.Request) {
	filter, errResult := parseIdFilter(r)
	if errResult != nil {
		WriteJson(w, errResult)
		return
	}

	if err := Downloader.Pause(filter); err != nil {
		WriteJson(w, model.NewErrorResult(err.Error()))
		return
	}
	WriteJson(w, model.NewNilResult())
}

func PauseTasks(w http.ResponseWriter, r *http.Request) {
	filter, errResult := parseFilter(r)
	if errResult != nil {
		WriteJson(w, errResult)
		return
	}

	if err := Downloader.Pause(filter); err != nil {
		WriteJson(w, model.NewErrorResult(err.Error()))
		return
	}
	WriteJson(w, model.NewNilResult())
}

func ContinueTask(w http.ResponseWriter, r *http.Request) {
	filter, errResult := parseIdFilter(r)
	if errResult != nil {
		WriteJson(w, errResult)
		return
	}

	if err := Downloader.Continue(filter); err != nil {
		WriteJson(w, model.NewErrorResult(err.Error()))
		return
	}
	WriteJson(w, model.NewNilResult())
}

func ContinueTasks(w http.ResponseWriter, r *http.Request) {
	filter, errResult := parseFilter(r)
	if errResult != nil {
		WriteJson(w, errResult)
		return
	}

	if err := Downloader.Continue(filter); err != nil {
		WriteJson(w, model.NewErrorResult(err.Error()))
		return
	}
	WriteJson(w, model.NewNilResult())
}

func DeleteTask(w http.ResponseWriter, r *http.Request) {
	filter, errResult := parseIdFilter(r)
	if errResult != nil {
		WriteJson(w, errResult)
		return
	}
	force := r.FormValue("force")

	if err := Downloader.Delete(filter, force == "true"); err != nil {
		WriteJson(w, model.NewErrorResult(err.Error()))
		return
	}
	WriteJson(w, model.NewNilResult())
}

func DeleteTasks(w http.ResponseWriter, r *http.Request) {
	filter, errResult := parseFilter(r)
	if errResult != nil {
		WriteJson(w, errResult)
		return
	}
	force := r.FormValue("force")

	if err := Downloader.Delete(filter, force == "true"); err != nil {
		WriteJson(w, model.NewErrorResult(err.Error()))
		return
	}
	WriteJson(w, model.NewNilResult())
}

func GetTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskId := vars["id"]
	if taskId == "" {
		WriteJson(w, model.NewErrorResult("param invalid: id", model.CodeInvalidParam))
		return
	}
	task := Downloader.GetTask(taskId)
	if task == nil {
		WriteJson(w, model.NewErrorResult("task not found", model.CodeTaskNotFound))
		return
	}
	WriteJson(w, model.NewOkResult(task))
}

func GetTasks(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("pageSize")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 {
		pageSize = 10
	}

	filter, errResult := parseFilter(r)
	if errResult != nil {
		WriteJson(w, errResult)
		return
	}

	tasks := Downloader.GetTasksByFilter(filter)
	totalTasks := len(tasks)
	start := (page - 1) * pageSize
	end := start + pageSize

	if start > totalTasks {
		start = totalTasks
	}
	if end > totalTasks {
		end = totalTasks
	}

	paginatedTasks := tasks[start:end]

	response := map[string]any{
		"tasks":      paginatedTasks,
		"total":      totalTasks,
		"page":       page,
		"pageSize":   pageSize,
		"totalPages": (totalTasks + pageSize - 1) / pageSize,
	}

	WriteJson(w, model.NewOkResult(response))
}

func GetConfig(w http.ResponseWriter, r *http.Request) {
	WriteJson(w, model.NewOkResult(getServerConfig()))
}

func PutConfig(w http.ResponseWriter, r *http.Request) {
	var cfg base.DownloaderStoreConfig
	if ReadJson(r, w, &cfg) {
		if err := Downloader.PutConfig(&cfg); err != nil {
			WriteJson(w, model.NewErrorResult(err.Error()))
			return
		}
	}
	WriteJson(w, model.NewNilResult())
}

func InstallExtension(w http.ResponseWriter, r *http.Request) {
	var req model.InstallExtension
	if ReadJson(r, w, &req) {
		var (
			installedExt *download.Extension
			err          error
		)
		if req.DevMode {
			installedExt, err = Downloader.InstallExtensionByFolder(req.URL, true)
		} else {
			installedExt, err = Downloader.InstallExtensionByGit(req.URL)
		}
		if err != nil {
			WriteJson(w, model.NewErrorResult(err.Error()))
			return
		}
		WriteJson(w, model.NewOkResult(installedExt.Identity))
	}
}

func GetExtensions(w http.ResponseWriter, r *http.Request) {
	list := Downloader.GetExtensions()
	WriteJson(w, model.NewOkResult(list))
}

func GetExtension(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	identity := vars["identity"]
	ext, err := Downloader.GetExtension(identity)
	if err != nil {
		WriteJson(w, model.NewErrorResult(err.Error()))
		return
	}
	WriteJson(w, model.NewOkResult(ext))
}

func UpdateExtensionSettings(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	identity := vars["identity"]
	var req model.UpdateExtensionSettings
	if ReadJson(r, w, &req) {
		if err := Downloader.UpdateExtensionSettings(identity, req.Settings); err != nil {
			WriteJson(w, model.NewErrorResult(err.Error()))
			return
		}
	}
	WriteJson(w, model.NewNilResult())
}

func SwitchExtension(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	identity := vars["identity"]
	var switchExtension model.SwitchExtension
	if ReadJson(r, w, &switchExtension) {
		if err := Downloader.SwitchExtension(identity, switchExtension.Status); err != nil {
			WriteJson(w, model.NewErrorResult(err.Error()))
			return
		}
	}
	WriteJson(w, model.NewNilResult())
}

func DeleteExtension(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	identity := vars["identity"]
	if err := Downloader.DeleteExtension(identity); err != nil {
		WriteJson(w, model.NewErrorResult(err.Error()))
		return
	}
	WriteJson(w, model.NewNilResult())
}

func UpdateCheckExtension(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	identity := vars["identity"]
	newVersion, err := Downloader.UpgradeCheckExtension(identity)
	if err != nil {
		WriteJson(w, model.NewErrorResult(err.Error()))
		return
	}
	WriteJson(w, model.NewOkResult(&model.UpdateCheckExtensionResp{
		NewVersion: newVersion,
	}))
}

func UpdateExtension(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	identity := vars["identity"]
	if err := Downloader.UpgradeExtension(identity); err != nil {
		WriteJson(w, model.NewErrorResult(err.Error()))
		return
	}
	WriteJson(w, model.NewNilResult())
}

func DoProxy(w http.ResponseWriter, r *http.Request) {
	target := r.Header.Get("X-Target-Uri")
	if target == "" {
		writeError(w, "param invalid: X-Target-Uri")
		return
	}
	targetUrl, err := url.Parse(target)
	if err != nil {
		writeError(w, err.Error())
		return
	}
	r.RequestURI = ""
	r.URL = targetUrl
	r.Host = targetUrl.Host
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		writeError(w, err.Error())
		return
	}
	defer resp.Body.Close()
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Set(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		writeError(w, err.Error())
		return
	}
	w.Write(buf)
}

func GetStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskId := vars["id"]
	if taskId == "" {
		WriteJson(w, model.NewErrorResult("param invalid: id", model.CodeInvalidParam))
		return
	}
	statsResult, err := Downloader.Stats(taskId)
	if err != nil {
		writeError(w, err.Error())
		return
	}
	WriteJson(w, model.NewOkResult(statsResult))
}

func parseIdFilter(r *http.Request) (*download.TaskFilter, any) {
	vars := mux.Vars(r)
	taskId := vars["id"]
	if taskId == "" {
		return nil, model.NewErrorResult("param invalid: id", model.CodeInvalidParam)
	}

	filter := &download.TaskFilter{
		IDs: []string{taskId},
	}
	return filter, nil
}

func parseFilter(r *http.Request) (*download.TaskFilter, any) {
	if err := r.ParseForm(); err != nil {
		return nil, model.NewErrorResult(err.Error())
	}

	filter := &download.TaskFilter{
		IDs:         r.Form["id"],
		Statuses:    convertStatues(r.Form["status"]),
		NotStatuses: convertStatues(r.Form["notStatus"]),
	}
	return filter, nil
}

func convertStatues(statues []string) []base.Status {
	result := make([]base.Status, 0)
	for _, status := range statues {
		result = append(result, base.Status(status))
	}
	return result
}

func writeError(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(msg))
}

func getServerConfig() *base.DownloaderStoreConfig {
	cfg, _ := Downloader.GetConfig()
	return cfg
}
