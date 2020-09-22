package mr

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
)

//noWorker assigned ass job's workerId when jobs has no worker
const noWorker = -1

//noReduceJobID assigned to job.reduceId when it has no reduce job yet
const noReduceJobID = -1

//JobType  what job worker has
type JobType string

const (
	mapJob    JobType = "mapJob"
	reduceJob         = "reduceJob"
	noJob             = "noJob"
)

//IsValid checks if given jobtype has valid string value
func (jt JobType) IsValid() bool {
	switch jt {
	case mapJob, reduceJob, noJob:
		return true
	}
	return false
}

//JobState represents in which state given job is
type JobState string

const (
	active   JobState = "active"
	done              = "done"
	loafting          = "loafting"
)

//IsValid checks if given jobstate has valid string value
func (js JobState) IsValid() bool {
	switch js {
	case active, done, loafting:
		return true
	}
	return false
}

//Job struct
type Job struct {
	ID       int
	workerID int
	files    []string
	jobState JobState
	JobType  JobType
	mapID    int
	reduceID int
}

//Master struct
type Master struct {
	nReduce    int
	files      []string
	jobs       map[int]Job
	tempFiles  []string
	nextWorker int        //next free worker id
	nextJob    int        // job id
	fMaps      int        //dinished maps
	fReduces   int        // finished reduces
	mlock      sync.Mutex // for ensuring fMaps valide value
	rlock      sync.Mutex // for ensuring fReduces valide value
	jlock      sync.Mutex // for locking jobs while handing out jobs
}

///for rpc communications

//HandOutJob hands out idle job to worker if it exists
func (m *Master) HandOutJob(
	arg *HandOutJobArg,
	res *HandOutJobResponse) error {
	m.jlock.Lock()

	for _, job := range m.jobs {
		fmt.Printf("%v\n", job.jobState)
		if job.jobState == loafting {
			res.Files = job.files
			res.JobID = job.ID
			res.NReduce = m.nReduce
			res.MapID = job.mapID
			res.JobType = job.JobType
			res.ReduceID = job.reduceID

			job.jobState = active
			job.workerID = arg.ID
			m.jobs[job.ID] = job

			break
		}
	}

	//unlock when done using jobs
	m.jlock.Unlock()

	return nil
}

//InitWorker tells worker it's id
func (m *Master) InitWorker(
	args *RPCEmptyArgument,
	reply *InitWorkerResponse) error {
	reply.ID = m.nextWorker
	m.nextWorker++
	return nil
}

//
// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
//
func (m *Master) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 2
	return nil
}

//
// start a thread that listens for RPCs from worker.go
//
func (m *Master) server() {
	rpc.Register(m)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := masterSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

//Done main/mrmaster.go calls Done() periodically to find out
// if the entire job has finished.
//
func (m *Master) Done() bool {
	ret := false

	// Your code here.

	return ret
}

func (m *Master) initJobs() {
	for _, file := range m.files {
		newJobID := m.nextJob
		m.nextJob++
		fmt.Printf("id %v\n", newJobID)
		newJob := Job{
			ID:       newJobID,
			JobType:  mapJob,
			files:    []string{file},
			jobState: loafting,
			workerID: noWorker,
			mapID:    newJobID,
			reduceID: noReduceJobID,
		}
		m.jobs[newJobID] = newJob

	}
}

//MakeMaster // create a Master.
// main/mrmaster.go calls this function.
// nReduce is the number of reduce tasks to use.
//
func MakeMaster(files []string, nReduce int) *Master {
	m := Master{
		tempFiles:  make([]string, 0),
		nextJob:    0,
		nextWorker: 0,
		jobs:       make(map[int]Job),
		fMaps:      0,
		fReduces:   0,
		mlock:      sync.Mutex{},
		rlock:      sync.Mutex{},
		jlock:      sync.Mutex{},
		nReduce:    nReduce,
		files:      files,
	}

	m.initJobs()

	m.server()
	return &m
}
