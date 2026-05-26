package osbuildexecutor

import (
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/osbuild/images/pkg/osbuild"
	"github.com/osbuild/osbuild-composer/internal/worker"
)

func pipelinePercent(done, total int) int {
	if total <= 0 {
		return 0
	}
	pct := (done * 100) / total
	if pct > 100 {
		pct = 100
	}
	return pct
}

func handleProgress(osbuildStatus *osbuild.StatusScanner, logger logrus.FieldLogger, job worker.Job) error {
	if osbuildStatus == nil {
		return fmt.Errorf("status scanner is required to handle osbuild progress")
	}

	var lastUpdated time.Time
	for {
		st, err := osbuildStatus.Status()
		if err != nil {
			return fmt.Errorf(`error parsing osbuild status, please report a bug: %w`, err)
		}
		if st == nil {
			break
		}

		progress := logrus.Fields{}
		if st.Progress != nil {
			progress["progress-done"] = st.Progress.Done
			progress["progress-total"] = st.Progress.Total
			if st.Progress.SubProgress != nil {
				progress["subprogress-done"] = st.Progress.SubProgress.Done
				progress["subprogress-total"] = st.Progress.SubProgress.Total
			}
		}
		if st.Message != "" {
			logger.WithFields(progress).Infof("OSBuild status: %s", st.Message)
			if job == nil || time.Since(lastUpdated) < MinTimeBetweenUpdates {
				continue
			}
			lastUpdated = time.Now()

			var partial worker.JobResult
			if strings.HasPrefix(st.Pipeline, "source") {
				partial = worker.JobResult{
					Progress: &worker.JobProgress{
						Message: "Preparing sources",
						Done:    0,
						Total:   100,
					},
				}
			} else {
				pct := 0
				if st.Progress != nil {
					pct = pipelinePercent(st.Progress.Done, st.Progress.Total)
				}
				partial = worker.JobResult{
					Progress: &worker.JobProgress{
						Message: "Building image",
						Done:    pct,
						Total:   100,
					},
				}
				if st.Progress != nil && st.Progress.SubProgress != nil {
					subMessage := st.Progress.SubProgress.Summary
					if subMessage == "" {
						subMessage = st.Progress.SubProgress.Message
					}
					if subMessage != "" {
						partial.Progress.SubProgress = &worker.JobProgress{
							Message: subMessage,
							Done:    st.Progress.SubProgress.Done,
							Total:   st.Progress.SubProgress.Total,
						}
					}
				}
			}
			err := job.Update(partial)
			if err != nil {
				logger.Errorf("Unable to update job: %s", err.Error())
			}
		}
		if st.Trace != "" {
			logger.Debugf("%s", st.Trace)
		}
	}
	return nil
}
