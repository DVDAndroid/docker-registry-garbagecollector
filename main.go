package main

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	docker "github.com/fsouza/go-dockerclient"
)

type Event struct {
	Action string `json:"action"`
	Target struct {
		Repository string `json:"repository"`
	}
}

type Events struct {
	Events []Event `json:"events"`
}

type Catalog struct {
	Repositories []string `json:"repositories"`
}

type TagsList struct {
	Tags []string `json:"tags"`
}

var dockerClient *docker.Client
var containerName string

func main() {
	PORT := os.Getenv("PORT")
	if PORT == "" {
		PORT = "5002"
	}
	containerName = os.Getenv("CONTAINER_NAME")

	var dockerErr error
	dockerClient, dockerErr = docker.NewClientFromEnv()
	if dockerErr != nil {
		slog.Error("failed to create docker client", "error", dockerErr)
		panic(dockerErr)
	}

	containers, err := dockerClient.ListContainers(docker.ListContainersOptions{
		Filters: map[string][]string{
			"name": {containerName},
		},
	})
	if err != nil {
		slog.Error("failed to list containers")
		panic(err)
	}
	if len(containers) == 0 {
		slog.Error("container not found", "container", containerName)
		panic("container not found")
	}

	http.HandleFunc("/", reqHandler)

	slog.Info("server is running", "port", PORT)
	httpError := http.ListenAndServe(":"+PORT, nil)
	if httpError != nil {
		slog.Error("failed to start server", httpError)
		panic(httpError)
	}
}

func reqHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		slog.Warn("invalid request method", "method", r.Method)
		w.WriteHeader(http.StatusOK) // return only 200
		return
	}

	var events Events
	err := json.NewDecoder(r.Body).Decode(&events)
	if err != nil {
		slog.Error("failed to decode request body", "error", err, "body", r.Body)
		w.WriteHeader(http.StatusOK) // return only 200
		return
	}

	// should be only one event
	var event = events.Events[0]
	// only get "delete" action

	if event.Action != "delete" {
		slog.Warn("invalid action", "action", event.Action)
		w.WriteHeader(http.StatusOK) // return only 200
		return
	}

	cleanupErr := cleanup(event.Target.Repository)
	if cleanupErr != nil {
		slog.Error("failed to cleanup", "error", cleanupErr)
		w.WriteHeader(http.StatusOK) // return only 200
		return
	}

	slog.Info("cleanup successful", "repository", event.Target.Repository)
	w.WriteHeader(http.StatusAccepted) // return 202
}

func cleanup(repository string) error {
	err := garbageCollect()
	if err != nil {
		slog.Error("failed to garbage collect", "error", err)
		return err
	}

	slog.Info("garbage collect successful", "repository", repository)

	err = diskCleanup(repository)
	if err != nil {
		slog.Error("failed to cleanup disk", "repository", repository, "error", err)
		return err
	}

	slog.Info("disk cleanup successful", "repository", repository)
	slog.Info("full cleanup successful", "repository", repository)
	return nil
}

func garbageCollect() error {
	execOptions := docker.CreateExecOptions{
		Container:    containerName,
		AttachStderr: true,
		AttachStdout: true,
		Cmd:          []string{"registry", "garbage-collect", "/etc/docker/registry/config.yml"},
	}
	execRes, err := dockerClient.CreateExec(execOptions)
	slog.Info("created exec", "id", execRes.ID)
	if err != nil {
		slog.Error("failed to create exec", "error", err)
		return err
	}

	var stdout, stderr bytes.Buffer
	startExecConfig := docker.StartExecOptions{
		OutputStream: &stdout,
		ErrorStream:  &stderr,
		RawTerminal:  true,
	}
	slog.Info("starting exec", "id", execRes.ID)
	err = dockerClient.StartExec(execRes.ID, startExecConfig)
	if err != nil {
		return err
	}

	slog.Info("printing stdout and stderr")
	slog.Info(stdout.String())
	slog.Error(stderr.String())
	slog.Info("exec successful")

	return nil
}

func diskCleanup(repository string) error {
	//send http req to http://CONTAINER_NAME:5000/v2/_catalog
	resp, err := http.Get("http://" + containerName + ":5000/v2/_catalog")
	if err != nil {
		slog.Error("failed to get catalog", "error", err)
		return err
	}
	defer resp.Body.Close()

	var catalog Catalog
	err = json.NewDecoder(resp.Body).Decode(&catalog)
	if err != nil {
		return err
	}

	var found bool
	for _, repo := range catalog.Repositories {
		if repo == repository {
			found = true
			break
		}
	}
	if !found {
		// already deleted
		return nil
	}

	// send http req to http://CONTAINER_NAME:5000/v2/REPOSITORY_NAME/tags/list
	resp, err = http.Get("http://" + containerName + ":5000/v2/" + repository + "/tags/list")
	if err != nil {
		slog.Error("failed to get tags list", "error", err)
		return err
	}
	defer resp.Body.Close()

	var tagsList TagsList
	err = json.NewDecoder(resp.Body).Decode(&tagsList)
	if err != nil {
		return err
	}
	var tags = tagsList.Tags

	// if tags != nil, do not cleanup
	if tags != nil {
		slog.Info("tags not empty, not cleaning up", "repository", repository, "tags", tags)
		return nil
	}

	// rm -rf /var/lib/registry/docker/registry/v2/repositories/REPOSITORY_NAME
	removeAllError := os.RemoveAll("/var/lib/registry/docker/registry/v2/repositories/" + repository)
	if removeAllError != nil {
		slog.Error("failed to remove directory", "error", removeAllError)
		return removeAllError
	}

	slog.Info("removed directory", "repository", repository)

	return nil
}
