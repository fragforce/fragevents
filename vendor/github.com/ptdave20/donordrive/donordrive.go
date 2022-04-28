package donordrive

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

const DefautlBaseUrl = "https://donordrive.com/"
const ExtraLifeUrl = "https://www.extra-life.org"
const TryBaseUrl = "https://try.donordrive.com/"

var baseUrl = DefautlBaseUrl
var client = &http.Client{}

const (
	apiEvents                = "api/events"
	apiEventsTeam            = "api/events/%d/teams"
	apiTeamParticipants      = "api/teams/%d/participants"
	apiTeamBadges            = "api/team/%d/badges"
	apiParticipantBadges     = "api/participants/%d/badges"
	apiParticipantDetails    = "api/participants/%d"
	apiParticipantDonations  = "api/participants/%d/donations"
	apiParticipantMilestones = "api/participants/%d/milestones"
	apiParticipantsTopDonor  = "api/participants/%d/donors"
)

func GetBaseUrl() string {
	return baseUrl
}

func SetBaseUrl(url string) {
	baseUrl = strings.TrimSpace(url)

	if strings.HasSuffix(baseUrl, "/") {
		return
	}

	baseUrl += "/"
}

func GetEvents() ([]Event, error) {
	res, err := client.Get(fmt.Sprintf("%s%s", baseUrl, apiEvents))
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("%d returned", res.StatusCode))
	}

	var results []Event

	decoder := json.NewDecoder(res.Body)
	if err := decoder.Decode(&results); err != nil {
		return nil, err
	}
	return results, nil
}

func GetTeamParticipants(team int) ([]Participant, error) {
	teamPath := fmt.Sprintf(apiTeamParticipants, team)
	res, err := client.Get(fmt.Sprintf("%s%s", baseUrl, teamPath))
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("%d returned", res.StatusCode))
	}

	var results []Participant

	decoder := json.NewDecoder(res.Body)
	if err := decoder.Decode(&results); err != nil {
		return nil, err
	}
	return results, nil
}

func GetParticipantDonations(participantID int) ([]Donation, error) {
	participantPath := fmt.Sprintf(apiParticipantDonations, participantID)
	res, err := client.Get(fmt.Sprintf("%s%s", baseUrl, participantPath))
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("%d returned", res.StatusCode))
	}

	var results []Donation

	decoder := json.NewDecoder(res.Body)
	if err := decoder.Decode(&results); err != nil {
		return nil, err
	}
	return results, nil
}

func GetParticipantMilestones(participantID int) ([]Milestone, error) {
	participantPath := fmt.Sprintf(apiParticipantMilestones, participantID)
	res, err := client.Get(fmt.Sprintf("%s%s", baseUrl, participantPath))
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("%d returned", res.StatusCode))
	}

	var results []Milestone

	decoder := json.NewDecoder(res.Body)
	if err := decoder.Decode(&results); err != nil {
		return nil, err
	}
	return results, nil
}

func GetParticipantDetails(participantID int) (*Participant, error) {
	participantPath := fmt.Sprintf(apiParticipantDetails, participantID)
	res, err := client.Get(fmt.Sprintf("%s%s", baseUrl, participantPath))
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("%d returned", res.StatusCode))
	}

	result := &Participant{}

	decoder := json.NewDecoder(res.Body)
	if err := decoder.Decode(result); err != nil {
		return nil, err
	}
	return result, nil
}

func GetParticipantBadges(participantID int) ([]Badge, error) {
	participantPath := fmt.Sprintf(apiParticipantBadges, participantID)
	res, err := client.Get(fmt.Sprintf("%s%s", baseUrl, participantPath))
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("%d returned", res.StatusCode))
	}

	var results []Badge

	decoder := json.NewDecoder(res.Body)
	if err := decoder.Decode(&results); err != nil {
		return nil, err
	}
	return results, nil
}
