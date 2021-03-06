package notification

import (
	"dt/events"
	"dt/models"
	"dt/utils"
	"dt/views"
	"encoding/json"
	"github.com/jinzhu/gorm"
)

type GroupJoinRequestDenied struct {
	notificationBase
	Request *models.GroupJoinRequest
} //notifier

func (req *GroupJoinRequestDenied) loadReceivers() {
	req.receivers = append(req.receivers, req.Request.InitiatorID)

	if *req.Request.AcceptorID != req.Request.Group.AdminID {
		req.receivers = append(req.receivers, req.Request.Group.AdminID)
	}
}

func (req *GroupJoinRequestDenied) loadDashReceivers() {
	for _, member := range req.Request.Group.Organization.Admins.Members {
		if member.UserID == *req.Request.AcceptorID {
			continue
		}

		req.dashReceivers = append(req.dashReceivers, member.UserID)
	}
}

func (req *GroupJoinRequestDenied) ContainerizedView() *utils.Container {
	return &utils.Container{
		Type: "notification.groupjoinrequestdenied",
		Data: req.View(),
	}
}

func (req *GroupJoinRequestDenied) View() interface{} {
	return &struct {
		ID      uint                    `json:"id"`
		Request *views.GroupJoinRequest `json:"request"`
		Seen    *bool                   `json:"seen,omitempty"`
	}{
		ID:      req.GetModel().ID,
		Request: views.GroupJoinRequestFromModelShort(req.Request),
		Seen:    req.seen,
	}
}

func (req *GroupJoinRequestDenied) CreateByEvent(db *gorm.DB, event interface{}) error {
	e, ok := event.(*events.GroupJoinRequestDenied)
	if !ok {
		return WrongEventErr
	}

	n, err := saveNotification(db, e)
	if err != nil {
		return err
	}

	if err = req.LoadWithEvent(db, e, n); err != nil {
		return err
	}

	wall, err := saveWallEvent(db, n, req.Request.Group.OrganizationID)
	if err != nil {
		return err
	}

	if _, err = saveAOWSExcept(db, wall, *req.Request.AcceptorID); err != nil {
		return err
	}

	if *req.Request.AcceptorID != req.Request.Group.AdminID {
		_, err = saveUNS(db, n, []uint{
			req.Request.Group.AdminID,
			req.Request.InitiatorID,
		})
	} else {
		_, err = saveSingleUNS(db, n, req.Request.InitiatorID)
	}

	if err != nil {
		return err
	}

	return nil
}

func (req *GroupJoinRequestDenied) Load(db *gorm.DB, n *models.Notification) error {
	var e *events.GroupJoinRequestDenied
	if err := json.Unmarshal(n.Data.RawMessage, &e); err != nil {
		return err
	}

	return req.LoadWithEvent(db, e, n)
}

func (req *GroupJoinRequestDenied) LoadWithEvent(
	db *gorm.DB,
	_event interface{},
	model *models.Notification,
) error {
	var event *events.GroupJoinRequestDenied
	var ok bool
	if event, ok = _event.(*events.GroupJoinRequestDenied); !ok {
		return WrongEventErr
	}

	var request models.GroupJoinRequest
	if err := db.First(&request, event.Request).Error; err != nil {
		return err
	}

	req.Request = &request
	req.Model = model
	req.loadReceivers()
	req.loadDashReceivers()

	return nil
}
