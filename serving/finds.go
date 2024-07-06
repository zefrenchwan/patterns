package serving

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/zefrenchwan/patterns.git/nodes"
	"github.com/zefrenchwan/patterns.git/storage"
)

func findElementFullPeriodHandler(wrapper ServiceParameters, w http.ResponseWriter, r *http.Request) error {
	return findElementDuringPeriodHandler(wrapper, w, r)
}

func findElementSinceHandler(wrapper ServiceParameters, w http.ResponseWriter, r *http.Request) error {
	return findElementDuringPeriodHandler(wrapper, w, r)
}

func findElementUntilHandler(wrapper ServiceParameters, w http.ResponseWriter, r *http.Request) error {
	return findElementDuringPeriodHandler(wrapper, w, r)
}

func findElementBetweenHandler(wrapper ServiceParameters, w http.ResponseWriter, r *http.Request) error {
	return findElementDuringPeriodHandler(wrapper, w, r)
}

func findElementDuringPeriodHandler(wrapper ServiceParameters, w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()

	user, auth := wrapper.CurrentUser()
	if !auth {
		return NewServiceForbiddenError("should authenticate")
	}

	minStr := r.PathValue("start")
	maxStr := r.PathValue("end")
	trait := r.PathValue("trait")

	// Build period
	var period nodes.Period
	var min, max time.Time
	if minStr != "" {
		if t, err := time.Parse(URL_DATE_FORMAT, minStr); err != nil {
			return NewServiceHttpClientError(err.Error())
		} else {
			min = t
		}
	}

	if maxStr != "" {
		if t, err := time.Parse(URL_DATE_FORMAT, maxStr); err != nil {
			return NewServiceHttpClientError(err.Error())
		} else {
			max = t
		}
	}

	switch {
	case minStr != "" && maxStr != "":
		interval, errInteval := nodes.NewFiniteTimeInterval(min, max, true, true)
		if errInteval != nil {
			return NewServiceHttpClientError(errInteval.Error())
		}

		period = nodes.NewPeriod(interval)
	case minStr != "":
		interval := nodes.NewRightInfiniteTimeInterval(min, true)
		period = nodes.NewPeriod(interval)
	case maxStr != "":
		interval := nodes.NewLeftInfiniteTimeInterval(max, true)
		period = nodes.NewPeriod(interval)
	default:
		period = nodes.NewFullPeriod()
	}

	// read parameters if any
	var globalErr error
	parameters := make(map[string]string)
	for value, elements := range r.URL.Query() {
		size := len(elements)
		if size == 0 {
			continue
		} else if size == 1 {
			parameters[value] = elements[0]
		} else {
			globalErr = errors.Join(globalErr, fmt.Errorf("too many parameters for %s", value))
		}
	}

	if globalErr != nil {
		return NewServiceHttpClientError(globalErr.Error())
	}

	// then, pass to dao
	graph, errLoad := wrapper.Dao.FindNeighborsOfMatchingElements(wrapper.Ctx, user, period, trait, parameters)
	if errLoad != nil {
		return NewServiceInternalServerError(errLoad.Error())
	}

	dto, errDto := storage.SerializeFullGraph(&graph, storage.SerializeElement)
	if errDto != nil {
		return NewServiceInternalServerError(errDto.Error())
	} else if err := json.NewEncoder(w).Encode(dto); err != nil {
		return NewServiceInternalServerError(err.Error())
	}

	return nil
}
