package jobs

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// JobState represents the state of a job
type JobState int

const (
	JobRunning JobState = iota
	JobStopped
	JobDone
)

func (s JobState) String() string {
	switch s {
	case JobRunning:
		return "Running"
	case JobStopped:
		return "Stopped"
	case JobDone:
		return "Done"
	default:
		return "Unknown"
	}
}

// Job represents a background job
type Job struct {
	ID        int
	PID       int
	PGID      int
	Command   string
	State     JobState
	Process   *os.Process
	ExitCode  int
	StartTime time.Time
}

// JobManager manages background jobs
type JobManager struct {
	jobs   map[int]*Job
	nextID int
	mutex  sync.RWMutex
}

// NewJobManager creates a new job manager
func NewJobManager() *JobManager {
	return &JobManager{
		jobs:   make(map[int]*Job),
		nextID: 1,
	}
}

// AddJob adds a new job to the manager
func (jm *JobManager) AddJob(cmd *exec.Cmd, command string) *Job {
	jm.mutex.Lock()
	defer jm.mutex.Unlock()

	job := &Job{
		ID:        jm.nextID,
		PID:       cmd.Process.Pid,
		PGID:      cmd.Process.Pid, // For simplicity, use PID as PGID
		Command:   command,
		State:     JobRunning,
		Process:   cmd.Process,
		StartTime: time.Now(),
	}

	jm.jobs[jm.nextID] = job
	jm.nextID++

	// Start monitoring the job
	go jm.monitorJob(job)

	return job
}

// GetJob returns a job by ID
func (jm *JobManager) GetJob(id int) *Job {
	jm.mutex.RLock()
	defer jm.mutex.RUnlock()
	return jm.jobs[id]
}

// GetJobs returns all jobs
func (jm *JobManager) GetJobs() []*Job {
	jm.mutex.RLock()
	defer jm.mutex.RUnlock()

	jobs := make([]*Job, 0, len(jm.jobs))
	for _, job := range jm.jobs {
		jobs = append(jobs, job)
	}
	return jobs
}

// GetActiveJobs returns only running and stopped jobs
func (jm *JobManager) GetActiveJobs() []*Job {
	jm.mutex.RLock()
	defer jm.mutex.RUnlock()

	jobs := make([]*Job, 0)
	for _, job := range jm.jobs {
		if job.State == JobRunning || job.State == JobStopped {
			jobs = append(jobs, job)
		}
	}
	return jobs
}

// RemoveJob removes a job from the manager
func (jm *JobManager) RemoveJob(id int) {
	jm.mutex.Lock()
	defer jm.mutex.Unlock()
	delete(jm.jobs, id)
}

// BringToForeground brings a job to the foreground
func (jm *JobManager) BringToForeground(id int) error {
	job := jm.GetJob(id)
	if job == nil {
		return fmt.Errorf("job %d not found", id)
	}

	if job.State == JobDone {
		return fmt.Errorf("job %d is already done", id)
	}

	// If the job is stopped, continue it
	if job.State == JobStopped {
		err := syscall.Kill(-job.PGID, syscall.SIGCONT)
		if err != nil {
			return fmt.Errorf("failed to continue job %d: %v", id, err)
		}
	}

	// Set job state to running
	jm.mutex.Lock()
	job.State = JobRunning
	jm.mutex.Unlock()

	// Wait for the job to complete
	state, err := job.Process.Wait()
	if err != nil {
		return fmt.Errorf("failed to wait for job %d: %v", id, err)
	}

	// Update job state
	jm.mutex.Lock()
	job.State = JobDone
	job.ExitCode = state.ExitCode()
	jm.mutex.Unlock()

	return nil
}

// SendToBackground sends a job to the background
func (jm *JobManager) SendToBackground(id int) error {
	job := jm.GetJob(id)
	if job == nil {
		return fmt.Errorf("job %d not found", id)
	}

	if job.State == JobDone {
		return fmt.Errorf("job %d is already done", id)
	}

	// If the job is stopped, continue it in the background
	if job.State == JobStopped {
		err := syscall.Kill(-job.PGID, syscall.SIGCONT)
		if err != nil {
			return fmt.Errorf("failed to continue job %d: %v", id, err)
		}

		jm.mutex.Lock()
		job.State = JobRunning
		jm.mutex.Unlock()
	}

	return nil
}

// StopJob stops a running job
func (jm *JobManager) StopJob(id int) error {
	job := jm.GetJob(id)
	if job == nil {
		return fmt.Errorf("job %d not found", id)
	}

	if job.State != JobRunning {
		return fmt.Errorf("job %d is not running", id)
	}

	// Send SIGSTOP to the process group
	err := syscall.Kill(-job.PGID, syscall.SIGSTOP)
	if err != nil {
		return fmt.Errorf("failed to stop job %d: %v", id, err)
	}

	jm.mutex.Lock()
	job.State = JobStopped
	jm.mutex.Unlock()

	return nil
}

// KillJob terminates a job
func (jm *JobManager) KillJob(id int) error {
	job := jm.GetJob(id)
	if job == nil {
		return fmt.Errorf("job %d not found", id)
	}

	if job.State == JobDone {
		return fmt.Errorf("job %d is already done", id)
	}

	// Send SIGTERM to the process group
	err := syscall.Kill(-job.PGID, syscall.SIGTERM)
	if err != nil {
		return fmt.Errorf("failed to kill job %d: %v", id, err)
	}

	jm.mutex.Lock()
	job.State = JobDone
	job.ExitCode = -1
	jm.mutex.Unlock()

	return nil
}

// monitorJob monitors a job for completion
func (jm *JobManager) monitorJob(job *Job) {
	state, err := job.Process.Wait()
	if err != nil {
		return
	}

	jm.mutex.Lock()
	defer jm.mutex.Unlock()

	job.State = JobDone
	job.ExitCode = state.ExitCode()
}

// CleanupDoneJobs removes completed jobs from the manager
func (jm *JobManager) CleanupDoneJobs() {
	jm.mutex.Lock()
	defer jm.mutex.Unlock()

	for id, job := range jm.jobs {
		if job.State == JobDone {
			delete(jm.jobs, id)
		}
	}
}

// PrintJobs prints all active jobs
func (jm *JobManager) PrintJobs() {
	jobs := jm.GetActiveJobs()
	if len(jobs) == 0 {
		return
	}

	for _, job := range jobs {
		status := job.State.String()
		if job.State == JobRunning {
			status = "Running"
		} else if job.State == JobStopped {
			status = "Stopped"
		}
		fmt.Printf("[%d]  %s\t\t%s\n", job.ID, status, job.Command)
	}
}
