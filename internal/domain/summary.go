package domain

func StatusCounts(recovery_jobs []RecoveryJob) map[RecoveryJobStatus]int {
	counts := make(map[RecoveryJobStatus]int)
	for _, task := range recovery_jobs {
		counts[task.Status]++
	}
	return counts
}
