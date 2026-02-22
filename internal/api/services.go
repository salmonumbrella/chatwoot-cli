package api

// Service accessors group Client methods by resource for gradual API slicing.
// Each service embeds *Client to avoid breaking existing call sites.

type AccountService struct{ *Client }

type AgentBotsService struct{ *Client }

type AgentsService struct{ *Client }

type AuditLogsService struct{ *Client }

type AutomationRulesService struct{ *Client }

type CampaignsService struct{ *Client }

type CannedResponsesService struct{ *Client }

type ContactsService struct{ *Client }

type ContextService struct{ *Client }

type ConversationsService struct{ *Client }

type CSATService struct{ *Client }

type CustomAttributesService struct{ *Client }

type CustomFiltersService struct{ *Client }

type InboxesService struct{ *Client }

type IntegrationsService struct{ *Client }

type LabelsService struct{ *Client }

type MentionsService struct{ *Client }

type MessagesService struct{ *Client }

type NotionService struct{ *Client }

type PlatformService struct{ *Client }

type PlatformAgentBotsService struct{ *Client }

type PortalsService struct{ *Client }

type ProfileService struct{ *Client }

type PublicService struct{ *Client }

type ReportsService struct{ *Client }

type ShopifyService struct{ *Client }

type SurveyService struct{ *Client }

type TeamsService struct{ *Client }

type WebhooksService struct{ *Client }

func (c *Client) Account() AccountService {
	return AccountService{c}
}

func (c *Client) AgentBots() AgentBotsService {
	return AgentBotsService{c}
}

func (c *Client) Agents() AgentsService {
	return AgentsService{c}
}

func (c *Client) AuditLogs() AuditLogsService {
	return AuditLogsService{c}
}

func (c *Client) AutomationRules() AutomationRulesService {
	return AutomationRulesService{c}
}

func (c *Client) Campaigns() CampaignsService {
	return CampaignsService{c}
}

func (c *Client) CannedResponses() CannedResponsesService {
	return CannedResponsesService{c}
}

func (c *Client) Contacts() ContactsService {
	return ContactsService{c}
}

func (c *Client) Context() ContextService {
	return ContextService{c}
}

func (c *Client) Conversations() ConversationsService {
	return ConversationsService{c}
}

func (c *Client) CSAT() CSATService {
	return CSATService{c}
}

func (c *Client) CustomAttributes() CustomAttributesService {
	return CustomAttributesService{c}
}

func (c *Client) CustomFilters() CustomFiltersService {
	return CustomFiltersService{c}
}

func (c *Client) Inboxes() InboxesService {
	return InboxesService{c}
}

func (c *Client) Integrations() IntegrationsService {
	return IntegrationsService{c}
}

func (c *Client) Labels() LabelsService {
	return LabelsService{c}
}

func (c *Client) Mentions() MentionsService {
	return MentionsService{c}
}

func (c *Client) Messages() MessagesService {
	return MessagesService{c}
}

func (c *Client) Notion() NotionService {
	return NotionService{c}
}

func (c *Client) Platform() PlatformService {
	return PlatformService{c}
}

func (c *Client) PlatformAgentBots() PlatformAgentBotsService {
	return PlatformAgentBotsService{c}
}

func (c *Client) Portals() PortalsService {
	return PortalsService{c}
}

func (c *Client) Profile() ProfileService {
	return ProfileService{c}
}

func (c *Client) Public() PublicService {
	return PublicService{c}
}

func (c *Client) Reports() ReportsService {
	return ReportsService{c}
}

func (c *Client) Shopify() ShopifyService {
	return ShopifyService{c}
}

func (c *Client) Survey() SurveyService {
	return SurveyService{c}
}

func (c *Client) Teams() TeamsService {
	return TeamsService{c}
}

func (c *Client) Webhooks() WebhooksService {
	return WebhooksService{c}
}
