package realtime

const (
	ActionCreated              = "created"
	ActionUpdated              = "updated"
	ActionDeleted              = "deleted"
	ActionCurrentClientChanged = "current_client_changed"
	ActionMemberAdded          = "member_added"
	ActionMemberRemoved        = "member_removed"
	ActionBackgroundUpdated    = "background_updated"
	ActionStatusChanged        = "status_changed"

	EntityUser     = "user"
	EntityClient   = "client"
	EntityCampaign = "campaign"
	EntityContent  = "content"
	EntityActivity = "activity"
)

func EventType(entity, action string) string {
	return entity + "." + action
}
