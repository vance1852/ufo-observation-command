package arrayops

func CloneCalibrationBundle(value CalibrationBundle) CalibrationBundle {
	clone := value
	clone.BuoyClasses = append([]string(nil), value.BuoyClasses...)
	clone.Labels = make(map[string]string, len(value.Labels))
	for key, item := range value.Labels {
		clone.Labels[key] = item
	}
	return clone
}

func CloneSurveyMission(value SurveyMission) SurveyMission {
	clone := value
	clone.BuoyIDs = append([]string(nil), value.BuoyIDs...)
	return clone
}

func CloneBuoys(values []Buoy) []Buoy {
	return append([]Buoy(nil), values...)
}

func RestoreLabels(snapshot map[string]string) map[string]string {
	labels := make(map[string]string, len(snapshot))
	for key, value := range snapshot {
		labels[key] = value
	}
	return labels
}
