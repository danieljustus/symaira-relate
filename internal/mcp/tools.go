package mcp

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/danieljustus/symaira-relate/internal/app"
	"github.com/danieljustus/symaira-relate/internal/domain/contact"
	"github.com/danieljustus/symaira-relate/internal/domain/page"
	"github.com/danieljustus/symaira-relate/internal/domain/security"
	"github.com/danieljustus/symaira-relate/internal/errs"
)

// tool is one MCP tool: its advertised schema plus the handler dispatched
// by tools/call. handler receives raw JSON arguments so each handler
// controls its own strict decoding and validation — an oversized or
// malformed argument object never reaches the service layer.
type tool struct {
	name        string
	description string
	inputSchema map[string]any
	handler     func(ctx context.Context, a *app.App, args json.RawMessage) (any, error)
}

func registerTools(s *Server) {
	for _, t := range []tool{
		contactSearchTool(),
		contactGetTool(),
		contactCreateTool(),
		contactUpdateTool(),
		organizationSearchTool(),
		organizationGetTool(),
		followUpListTool(),
		timelineGetTool(),
	} {
		s.tools[t.name] = t
	}
}

func (s *Server) handleToolsList() any {
	names := make([]string, 0, len(s.tools))
	for name := range s.tools {
		names = append(names, name)
	}
	sortStrings(names)

	list := make([]map[string]any, 0, len(names))
	for _, name := range names {
		t := s.tools[name]
		list = append(list, map[string]any{
			"name":        t.name,
			"description": t.description,
			"inputSchema": t.inputSchema,
		})
	}
	return map[string]any{"tools": list}
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}

type toolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

func (s *Server) handleToolsCall(ctx context.Context, params json.RawMessage) (any, error) {
	var p toolCallParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, errs.Invalid("mcp.tools/call", "invalid params: expected {name, arguments}", err)
	}
	t, ok := s.tools[p.Name]
	if !ok {
		return nil, errs.Invalid("mcp.tools/call", "unknown tool: "+p.Name, nil)
	}

	result, err := t.handler(ctx, s.app, p.Arguments)
	if err != nil {
		return toolCallResult(nil, err), nil // tool-call errors are reported in-band, not as an RPC error
	}
	return toolCallResult(result, nil), nil
}

// toolCallResult wraps a tool's return value in the MCP tool-call result
// shape: a text content block (for clients that only render text) plus
// structuredContent carrying the same value as real JSON (for clients
// that parse it directly), matching the CLI's JSON contract 1:1 — see
// docs/MCP.md.
func toolCallResult(result any, callErr error) map[string]any {
	if callErr != nil {
		return map[string]any{
			"isError": true,
			"content": []map[string]any{{
				"type": "text",
				"text": redactedMessage(callErr),
			}},
		}
	}
	text, _ := json.MarshalIndent(result, "", "  ")
	return map[string]any{
		"isError":           false,
		"content":           []map[string]any{{"type": "text", "text": string(text)}},
		"structuredContent": result,
	}
}

// -- contact_search ------------------------------------------------------

func contactSearchTool() tool {
	return tool{
		name: "contact_search",
		description: "Search and list people by name substring and/or classification, paginated. Read-only. " +
			"Never use this to bulk-ingest contacts into another system — results are for the calling agent's " +
			"immediate task only.",
		inputSchema: objectSchema(map[string]any{
			"query":          stringProp("Case-insensitive substring matched against display/given/family name"),
			"classification": stringProp("Filter: personal, business, customer, lead, or partner"),
			"limit":          intProp("Max results (server-bounded; see docs/CLI_CONTRACT.md)"),
			"offset":         intProp("Result offset for pagination"),
		}, nil),
		handler: func(ctx context.Context, a *app.App, raw json.RawMessage) (any, error) {
			var args struct {
				Query          string `json:"query"`
				Classification string `json:"classification"`
				Limit          int    `json:"limit"`
				Offset         int    `json:"offset"`
			}
			if err := decodeArgs(raw, &args); err != nil {
				return nil, err
			}
			return a.Contacts.ListPersons(ctx, app.ListPersonsOptions{
				Classification: contact.Classification(args.Classification),
				Query:          args.Query,
				Page:           page.Request{Limit: args.Limit, Offset: args.Offset},
			})
		},
	}
}

// -- contact_get -----------------------------------------------------------

func contactGetTool() tool {
	return tool{
		name:        "contact_get",
		description: "Load one person by id, including contact points, aliases, tags and classifications. Read-only. Never expose sensitive contact-point values beyond what this call explicitly returns.",
		inputSchema: objectSchema(map[string]any{
			"id": stringProp("Person id"),
		}, []string{"id"}),
		handler: func(ctx context.Context, a *app.App, raw json.RawMessage) (any, error) {
			var args struct {
				ID string `json:"id"`
			}
			if err := decodeArgs(raw, &args); err != nil {
				return nil, err
			}
			if args.ID == "" {
				return nil, errs.Invalid("mcp.contact_get", "id is required", nil)
			}
			return a.Contacts.GetPerson(ctx, args.ID)
		},
	}
}

// -- contact_create --------------------------------------------------------

func contactCreateTool() tool {
	return tool{
		name: "contact_create",
		description: "Create a new person. This is an explicit, deliberate mutation — never call this to auto-ingest " +
			"contacts discovered incidentally (a meeting attendee, an email sender); only call it when the user has " +
			"asked to add this specific person.",
		inputSchema: objectSchema(map[string]any{
			"display_name": stringProp("Display name (required)"),
			"given_name":   stringProp("Given name"),
			"family_name":  stringProp("Family name"),
			"notes":        stringProp("Free-text notes"),
			"email":        stringProp("Email contact point"),
			"phone":        stringProp("Phone contact point"),
		}, []string{"display_name"}),
		handler: func(ctx context.Context, a *app.App, raw json.RawMessage) (any, error) {
			var args struct {
				DisplayName string `json:"display_name"`
				GivenName   string `json:"given_name"`
				FamilyName  string `json:"family_name"`
				Notes       string `json:"notes"`
				Email       string `json:"email"`
				Phone       string `json:"phone"`
			}
			if err := decodeArgs(raw, &args); err != nil {
				return nil, err
			}
			return a.CreatePersonWithContactPoints(ctx, contact.PersonInput{
				DisplayName: args.DisplayName, GivenName: args.GivenName, FamilyName: args.FamilyName, Notes: args.Notes,
			}, args.Email, args.Phone)
		},
	}
}

// -- contact_update --------------------------------------------------------

func contactUpdateTool() tool {
	return tool{
		name:        "contact_update",
		description: "Patch an existing person's display name and/or notes. Only fields provided are changed. Explicit, deliberate mutation only.",
		inputSchema: objectSchema(map[string]any{
			"id":           stringProp("Person id (required)"),
			"display_name": stringProp("New display name"),
			"notes":        stringProp("New notes"),
		}, []string{"id"}),
		handler: func(ctx context.Context, a *app.App, raw json.RawMessage) (any, error) {
			var args struct {
				ID          string  `json:"id"`
				DisplayName *string `json:"display_name"`
				Notes       *string `json:"notes"`
			}
			if err := decodeArgs(raw, &args); err != nil {
				return nil, err
			}
			if args.ID == "" {
				return nil, errs.Invalid("mcp.contact_update", "id is required", nil)
			}
			return a.Contacts.UpdatePerson(ctx, args.ID, contact.PersonUpdate{
				DisplayName: args.DisplayName, Notes: args.Notes,
			})
		},
	}
}

// -- organization_search / organization_get --------------------------------

func organizationSearchTool() tool {
	return tool{
		name:        "organization_search",
		description: "Search and list organizations by name substring and/or classification, paginated. Read-only.",
		inputSchema: objectSchema(map[string]any{
			"query":          stringProp("Case-insensitive substring matched against the organization name"),
			"classification": stringProp("Filter: personal, business, customer, lead, or partner"),
			"limit":          intProp("Max results (server-bounded)"),
			"offset":         intProp("Result offset for pagination"),
		}, nil),
		handler: func(ctx context.Context, a *app.App, raw json.RawMessage) (any, error) {
			var args struct {
				Query          string `json:"query"`
				Classification string `json:"classification"`
				Limit          int    `json:"limit"`
				Offset         int    `json:"offset"`
			}
			if err := decodeArgs(raw, &args); err != nil {
				return nil, err
			}
			return a.Contacts.ListOrganizations(ctx, app.ListOrganizationsOptions{
				Classification: contact.Classification(args.Classification),
				Query:          args.Query,
				Page:           page.Request{Limit: args.Limit, Offset: args.Offset},
			})
		},
	}
}

func organizationGetTool() tool {
	return tool{
		name:        "organization_get",
		description: "Load one organization by id, including contact points, aliases, tags and classifications. Read-only.",
		inputSchema: objectSchema(map[string]any{
			"id": stringProp("Organization id"),
		}, []string{"id"}),
		handler: func(ctx context.Context, a *app.App, raw json.RawMessage) (any, error) {
			var args struct {
				ID string `json:"id"`
			}
			if err := decodeArgs(raw, &args); err != nil {
				return nil, err
			}
			if args.ID == "" {
				return nil, errs.Invalid("mcp.organization_get", "id is required", nil)
			}
			return a.Contacts.GetOrganization(ctx, args.ID)
		},
	}
}

// -- follow_up_list ----------------------------------------------------------

func followUpListTool() tool {
	return tool{
		name: "followup_list",
		description: "List follow-up reminders for a person or organization, optionally filtered to open/overdue/" +
			"upcoming. Read-only.",
		inputSchema: objectSchema(map[string]any{
			"person_id":       stringProp("Person id — exactly one of person_id/organization_id is required"),
			"organization_id": stringProp("Organization id — exactly one of person_id/organization_id is required"),
			"filter":          stringProp("all (default), open, overdue, or upcoming"),
		}, nil),
		handler: func(ctx context.Context, a *app.App, raw json.RawMessage) (any, error) {
			var args struct {
				PersonID       string `json:"person_id"`
				OrganizationID string `json:"organization_id"`
				Filter         string `json:"filter"`
			}
			if err := decodeArgs(raw, &args); err != nil {
				return nil, err
			}
			if (args.PersonID == "") == (args.OrganizationID == "") {
				return nil, errs.Invalid("mcp.followup_list", "exactly one of person_id or organization_id is required", nil)
			}
			filter := app.FollowUpFilter(args.Filter)
			if filter == "" {
				filter = app.FollowUpFilterAll
			}
			if args.PersonID != "" {
				return a.Relationships.ListPersonFollowUps(ctx, args.PersonID, filter)
			}
			return a.Relationships.ListOrganizationFollowUps(ctx, args.OrganizationID, filter)
		},
	}
}

// -- timeline_get ------------------------------------------------------------

func timelineGetTool() tool {
	return tool{
		name:        "timeline_get",
		description: "Get a person's or organization's combined interaction and follow-up timeline, most recent first. Read-only.",
		inputSchema: objectSchema(map[string]any{
			"person_id":       stringProp("Person id — exactly one of person_id/organization_id is required"),
			"organization_id": stringProp("Organization id — exactly one of person_id/organization_id is required"),
		}, nil),
		handler: func(ctx context.Context, a *app.App, raw json.RawMessage) (any, error) {
			var args struct {
				PersonID       string `json:"person_id"`
				OrganizationID string `json:"organization_id"`
			}
			if err := decodeArgs(raw, &args); err != nil {
				return nil, err
			}
			if (args.PersonID == "") == (args.OrganizationID == "") {
				return nil, errs.Invalid("mcp.timeline_get", "exactly one of person_id or organization_id is required", nil)
			}
			if args.PersonID != "" {
				return a.Relationships.PersonTimeline(ctx, args.PersonID)
			}
			return a.Relationships.OrganizationTimeline(ctx, args.OrganizationID)
		},
	}
}

// -- shared helpers ----------------------------------------------------------

func decodeArgs(raw json.RawMessage, dst any) error {
	if len(raw) == 0 {
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return errs.Invalid("mcp.decodeArgs", "invalid arguments: "+err.Error(), err)
	}
	return nil
}

func objectSchema(properties map[string]any, required []string) map[string]any {
	s := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		s["required"] = required
	}
	return s
}

func stringProp(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}

func intProp(description string) map[string]any {
	return map[string]any{"type": "integer", "description": description}
}

// redactedMessage renders a tool-call error's text content block. Tool-call
// errors are reported in-band (isError: true) rather than as an RPC error
// so a client can distinguish "the tool ran and reported failure" from
// "the transport itself failed" — the message is still redacted, exactly
// like the RPC error path in errorToResponse.
func redactedMessage(err error) string {
	return security.Redact(err.Error())
}
