package api

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/services/accesscontrol"
	"github.com/grafana/grafana/pkg/setting"
)

var (
	searchOrgsURL           = "/api/orgs/"
	getCurrentOrgURL        = "/api/org/"
	getOrgsURL              = "/api/orgs/%v"
	getOrgsByNameURL        = "/api/orgs/name/%v"
	putCurrentOrgURL        = "/api/org/"
	putOrgsURL              = "/api/orgs/%v"
	putCurrentOrgAddressURL = "/api/org/address"
	putOrgsAddressURL       = "/api/orgs/%v/address"

	testUpdateOrgNameForm    = `{ "name": "TestOrgChanged" }`
	testUpdateOrgAddressForm = `{ "address1": "1 test road",
	"address2": "2 test road",
	"city": "TestCity",
	"ZipCode": "TESTZIPCODE",
	"State": "TestState",
	"Country": "TestCountry" }`

	deleteOrgsURL = "/api/orgs/%v"

	createOrgsURL    = "/api/orgs/"
	testCreateOrgCmd = `{ "name": "TestOrg%v"}`
)

func TestAPIEndpoint_CreateOrgs_LegacyAccessControl(t *testing.T) {
	sc := setupHTTPServer(t, false)
	setInitCtxSignedInViewer(sc.initCtx)

	_, err := sc.db.CreateOrgWithMember("TestOrg", testUserID)
	require.NoError(t, err)

	setting.AllowUserOrgCreate = false
	input := strings.NewReader(fmt.Sprintf(testCreateOrgCmd, 2))
	t.Run("Viewer cannot create Orgs", func(t *testing.T) {
		response := callAPI(sc.server, http.MethodPost, createOrgsURL, input, t)
		assert.Equal(t, http.StatusForbidden, response.Code)
	})

	sc.initCtx.SignedInUser.IsGrafanaAdmin = true
	input = strings.NewReader(fmt.Sprintf(testCreateOrgCmd, 3))
	t.Run("Grafana Admin viewer can create Orgs", func(t *testing.T) {
		response := callAPI(sc.server, http.MethodPost, createOrgsURL, input, t)
		assert.Equal(t, http.StatusOK, response.Code)
	})

	sc.initCtx.SignedInUser.IsGrafanaAdmin = false
	setting.AllowUserOrgCreate = true
	input = strings.NewReader(fmt.Sprintf(testCreateOrgCmd, 4))
	t.Run("User viewer can create Orgs when AllowUserOrgCreate setting is true", func(t *testing.T) {
		response := callAPI(sc.server, http.MethodPost, createOrgsURL, input, t)
		assert.Equal(t, http.StatusOK, response.Code)
	})
}

func TestAPIEndpoint_CreateOrgs_AccessControl(t *testing.T) {
	sc := setupHTTPServer(t, true)
	setInitCtxSignedInViewer(sc.initCtx)

	_, err := sc.db.CreateOrgWithMember("TestOrg", testUserID)
	require.NoError(t, err)

	input := strings.NewReader(fmt.Sprintf(testCreateOrgCmd, 2))
	t.Run("AccessControl allows creating Orgs with correct permissions", func(t *testing.T) {
		setAccessControlPermissions(sc.acmock, []*accesscontrol.Permission{{Action: ActionOrgsCreate}})
		response := callAPI(sc.server, http.MethodPost, createOrgsURL, input, t)
		assert.Equal(t, http.StatusOK, response.Code)
	})

	input = strings.NewReader(fmt.Sprintf(testCreateOrgCmd, 3))
	t.Run("AccessControl prevents creating Orgs with incorrect permissions", func(t *testing.T) {
		setAccessControlPermissions(sc.acmock, []*accesscontrol.Permission{{Action: "orgs:invalid"}})
		response := callAPI(sc.server, http.MethodPost, createOrgsURL, input, t)
		assert.Equal(t, http.StatusForbidden, response.Code)
	})
}

func TestAPIEndpoint_DeleteOrgs_LegacyAccessControl(t *testing.T) {
	sc := setupHTTPServer(t, false)
	setInitCtxSignedInViewer(sc.initCtx)

	// Create two orgs
	_, err := sc.db.CreateOrgWithMember("TestOrg", testUserID)
	require.NoError(t, err)
	_, err = sc.db.CreateOrgWithMember("TestOrg2", testUserID)
	require.NoError(t, err)

	t.Run("Viewer cannot delete Orgs", func(t *testing.T) {
		response := callAPI(sc.server, http.MethodDelete, fmt.Sprintf(deleteOrgsURL, 2), nil, t)
		assert.Equal(t, http.StatusForbidden, response.Code)
	})

	sc.initCtx.SignedInUser.IsGrafanaAdmin = true
	t.Run("Grafana Admin viewer can delete Orgs", func(t *testing.T) {
		response := callAPI(sc.server, http.MethodDelete, fmt.Sprintf(deleteOrgsURL, 2), nil, t)
		assert.Equal(t, http.StatusOK, response.Code)
	})
}

func TestAPIEndpoint_DeleteOrgs_AccessControl(t *testing.T) {
	sc := setupHTTPServer(t, true)
	setInitCtxSignedInViewer(sc.initCtx)

	// Create three orgs (to delete org2 then org3)
	_, err := sc.db.CreateOrgWithMember("TestOrg", testUserID)
	require.NoError(t, err)
	_, err = sc.db.CreateOrgWithMember("TestOrg2", testUserID)
	require.NoError(t, err)
	_, err = sc.db.CreateOrgWithMember("TestOrg3", testUserID)
	require.NoError(t, err)

	t.Run("AccessControl allows deleting Orgs with correct permissions", func(t *testing.T) {
		setAccessControlPermissions(sc.acmock, []*accesscontrol.Permission{{Action: ActionOrgsDelete, Scope: ScopeOrgsAll}})
		response := callAPI(sc.server, http.MethodDelete, fmt.Sprintf(deleteOrgsURL, 2), nil, t)
		assert.Equal(t, http.StatusOK, response.Code)
	})
	t.Run("AccessControl allows deleting Orgs with exact permissions", func(t *testing.T) {
		setAccessControlPermissions(sc.acmock, []*accesscontrol.Permission{{Action: ActionOrgsDelete, Scope: accesscontrol.Scope("orgs", "id", "3")}})
		response := callAPI(sc.server, http.MethodDelete, fmt.Sprintf(deleteOrgsURL, 3), nil, t)
		assert.Equal(t, http.StatusOK, response.Code)
	})
	t.Run("AccessControl prevents deleting Orgs with too narrow permissions", func(t *testing.T) {
		setAccessControlPermissions(sc.acmock, []*accesscontrol.Permission{{Action: ActionOrgsDelete, Scope: accesscontrol.Scope("orgs", "id", "1")}})
		response := callAPI(sc.server, http.MethodDelete, fmt.Sprintf(deleteOrgsURL, 2), nil, t)
		assert.Equal(t, http.StatusForbidden, response.Code)
	})
	t.Run("AccessControl prevents deleting Orgs with incorrect permissions", func(t *testing.T) {
		setAccessControlPermissions(sc.acmock, []*accesscontrol.Permission{{Action: "orgs:invalid"}})
		response := callAPI(sc.server, http.MethodDelete, fmt.Sprintf(deleteOrgsURL, 2), nil, t)
		assert.Equal(t, http.StatusForbidden, response.Code)
	})
}

func TestAPIEndpoint_SearchOrgs_LegacyAccessControl(t *testing.T) {
	sc := setupHTTPServer(t, false)
	setInitCtxSignedInViewer(sc.initCtx)

	// Create two orgs
	_, err := sc.db.CreateOrgWithMember("TestOrg", testUserID)
	require.NoError(t, err)
	_, err = sc.db.CreateOrgWithMember("TestOrg2", testUserID)
	require.NoError(t, err)

	t.Run("Viewer cannot list Orgs", func(t *testing.T) {
		response := callAPI(sc.server, http.MethodGet, searchOrgsURL, nil, t)
		assert.Equal(t, http.StatusForbidden, response.Code)
	})

	sc.initCtx.SignedInUser.IsGrafanaAdmin = true
	t.Run("Grafana Admin viewer can list Orgs", func(t *testing.T) {
		response := callAPI(sc.server, http.MethodGet, searchOrgsURL, nil, t)
		assert.Equal(t, http.StatusOK, response.Code)
	})
}

func TestAPIEndpoint_SearchOrgs_AccessControl(t *testing.T) {
	sc := setupHTTPServer(t, true)
	setInitCtxSignedInViewer(sc.initCtx)

	// Create two orgs
	_, err := sc.db.CreateOrgWithMember("TestOrg", testUserID)
	require.NoError(t, err)
	_, err = sc.db.CreateOrgWithMember("TestOrg2", testUserID)
	require.NoError(t, err)

	t.Run("AccessControl allows listing Orgs with correct permissions", func(t *testing.T) {
		setAccessControlPermissions(sc.acmock, []*accesscontrol.Permission{{Action: ActionOrgsRead, Scope: ScopeOrgsAll}})
		response := callAPI(sc.server, http.MethodGet, searchOrgsURL, nil, t)
		assert.Equal(t, http.StatusOK, response.Code)
	})
	t.Run("AccessControl prevents listing Orgs with too narrow permissions", func(t *testing.T) {
		setAccessControlPermissions(sc.acmock, []*accesscontrol.Permission{{Action: ActionOrgsRead, Scope: accesscontrol.Scope("orgs", "id", "1")}})
		response := callAPI(sc.server, http.MethodGet, searchOrgsURL, nil, t)
		assert.Equal(t, http.StatusForbidden, response.Code)
	})
	t.Run("AccessControl prevents listing Orgs with incorrect permissions", func(t *testing.T) {
		setAccessControlPermissions(sc.acmock, []*accesscontrol.Permission{{Action: "orgs:invalid"}})
		response := callAPI(sc.server, http.MethodGet, searchOrgsURL, nil, t)
		assert.Equal(t, http.StatusForbidden, response.Code)
	})
}

func TestAPIEndpoint_GetCurrentOrg_LegacyAccessControl(t *testing.T) {
	sc := setupHTTPServer(t, false)
	setInitCtxSignedInViewer(sc.initCtx)

	_, err := sc.db.CreateOrgWithMember("TestOrg", testUserID)
	require.NoError(t, err)

	t.Run("Viewer can view CurrentOrg", func(t *testing.T) {
		response := callAPI(sc.server, http.MethodGet, getCurrentOrgURL, nil, t)
		assert.Equal(t, http.StatusOK, response.Code)
	})

	sc.initCtx.IsSignedIn = false
	t.Run("Unsigned user cannot view CurrentOrg", func(t *testing.T) {
		response := callAPI(sc.server, http.MethodGet, getCurrentOrgURL, nil, t)
		assert.Equal(t, http.StatusUnauthorized, response.Code)
	})
}

func TestAPIEndpoint_GetCurrentOrg_AccessControl(t *testing.T) {
	sc := setupHTTPServer(t, true)
	setInitCtxSignedInViewer(sc.initCtx)

	_, err := sc.db.CreateOrgWithMember("TestOrg", testUserID)
	require.NoError(t, err)

	t.Run("AccessControl allows viewing CurrentOrg with correct permissions", func(t *testing.T) {
		setAccessControlPermissions(sc.acmock, []*accesscontrol.Permission{{Action: ActionOrgsRead, Scope: ScopeOrgsAll}})
		response := callAPI(sc.server, http.MethodGet, getCurrentOrgURL, nil, t)
		assert.Equal(t, http.StatusOK, response.Code)
	})
	t.Run("AccessControl allows viewing CurrentOrg with exact permissions", func(t *testing.T) {
		setAccessControlPermissions(sc.acmock, []*accesscontrol.Permission{{Action: ActionOrgsRead, Scope: accesscontrol.Scope("orgs", "id", "1")}})
		response := callAPI(sc.server, http.MethodGet, getCurrentOrgURL, nil, t)
		assert.Equal(t, http.StatusOK, response.Code)
	})
	t.Run("AccessControl prevents viewing CurrentOrg with incorrect permissions", func(t *testing.T) {
		setAccessControlPermissions(sc.acmock, []*accesscontrol.Permission{{Action: "orgs:invalid"}})
		response := callAPI(sc.server, http.MethodGet, getCurrentOrgURL, nil, t)
		assert.Equal(t, http.StatusForbidden, response.Code)
	})
}

func TestAPIEndpoint_GetOrg_LegacyAccessControl(t *testing.T) {
	sc := setupHTTPServer(t, false)
	setInitCtxSignedInViewer(sc.initCtx)

	// Create two orgs, to fetch another one than the logged in one
	_, err := sc.db.CreateOrgWithMember("TestOrg", testUserID)
	require.NoError(t, err)
	_, err = sc.db.CreateOrgWithMember("TestOrg2", testUserID)
	require.NoError(t, err)

	t.Run("Viewer cannot view another Org", func(t *testing.T) {
		response := callAPI(sc.server, http.MethodGet, fmt.Sprintf(getOrgsURL, 2), nil, t)
		assert.Equal(t, http.StatusForbidden, response.Code)
	})

	sc.initCtx.SignedInUser.IsGrafanaAdmin = true
	t.Run("Grafana admin viewer can view another Org", func(t *testing.T) {
		response := callAPI(sc.server, http.MethodGet, fmt.Sprintf(getOrgsURL, 2), nil, t)
		assert.Equal(t, http.StatusOK, response.Code)
	})
}

func TestAPIEndpoint_GetOrg_AccessControl(t *testing.T) {
	sc := setupHTTPServer(t, true)
	setInitCtxSignedInViewer(sc.initCtx)

	// Create two orgs, to fetch another one than the logged in one
	_, err := sc.db.CreateOrgWithMember("TestOrg", testUserID)
	require.NoError(t, err)
	_, err = sc.db.CreateOrgWithMember("TestOrg2", testUserID)
	require.NoError(t, err)

	t.Run("AccessControl allows viewing another org with correct permissions", func(t *testing.T) {
		setAccessControlPermissions(sc.acmock, []*accesscontrol.Permission{{Action: ActionOrgsRead, Scope: ScopeOrgsAll}})
		response := callAPI(sc.server, http.MethodGet, fmt.Sprintf(getOrgsURL, 2), nil, t)
		assert.Equal(t, http.StatusOK, response.Code)
	})
	t.Run("AccessControl allows viewing another org with exact permissions", func(t *testing.T) {
		setAccessControlPermissions(sc.acmock, []*accesscontrol.Permission{{Action: ActionOrgsRead, Scope: accesscontrol.Scope("orgs", "id", "2")}})
		response := callAPI(sc.server, http.MethodGet, fmt.Sprintf(getOrgsURL, 2), nil, t)
		assert.Equal(t, http.StatusOK, response.Code)
	})
	t.Run("AccessControl prevents viewing another org with too narrow permissions", func(t *testing.T) {
		setAccessControlPermissions(sc.acmock, []*accesscontrol.Permission{{Action: ActionOrgsRead, Scope: accesscontrol.Scope("orgs", "id", "1")}})
		response := callAPI(sc.server, http.MethodGet, fmt.Sprintf(getOrgsURL, 2), nil, t)
		assert.Equal(t, http.StatusForbidden, response.Code)
	})
	t.Run("AccessControl prevents viewing another org with incorrect permissions", func(t *testing.T) {
		setAccessControlPermissions(sc.acmock, []*accesscontrol.Permission{{Action: "orgs:invalid"}})
		response := callAPI(sc.server, http.MethodGet, fmt.Sprintf(getOrgsURL, 2), nil, t)
		assert.Equal(t, http.StatusForbidden, response.Code)
	})
}

func TestAPIEndpoint_GetOrgByName_LegacyAccessControl(t *testing.T) {
	sc := setupHTTPServer(t, false)
	setInitCtxSignedInViewer(sc.initCtx)

	// Create two orgs, to fetch another one than the logged in one
	_, err := sc.db.CreateOrgWithMember("TestOrg", testUserID)
	require.NoError(t, err)
	_, err = sc.db.CreateOrgWithMember("TestOrg2", testUserID)
	require.NoError(t, err)

	t.Run("Viewer cannot view another Org", func(t *testing.T) {
		response := callAPI(sc.server, http.MethodGet, fmt.Sprintf(getOrgsByNameURL, "TestOrg2"), nil, t)
		assert.Equal(t, http.StatusForbidden, response.Code)
	})

	sc.initCtx.SignedInUser.IsGrafanaAdmin = true
	t.Run("Grafana admin viewer can view another Org", func(t *testing.T) {
		response := callAPI(sc.server, http.MethodGet, fmt.Sprintf(getOrgsByNameURL, "TestOrg2"), nil, t)
		assert.Equal(t, http.StatusOK, response.Code)
	})
}

func TestAPIEndpoint_GetOrgByName_AccessControl(t *testing.T) {
	sc := setupHTTPServer(t, true)
	setInitCtxSignedInViewer(sc.initCtx)

	// Create two orgs, to fetch another one than the logged in one
	_, err := sc.db.CreateOrgWithMember("TestOrg", testUserID)
	require.NoError(t, err)
	_, err = sc.db.CreateOrgWithMember("TestOrg2", testUserID)
	require.NoError(t, err)

	t.Run("AccessControl allows viewing another org with correct permissions", func(t *testing.T) {
		setAccessControlPermissions(sc.acmock, []*accesscontrol.Permission{{Action: ActionOrgsRead, Scope: ScopeOrgsAll}})
		response := callAPI(sc.server, http.MethodGet, fmt.Sprintf(getOrgsByNameURL, "TestOrg2"), nil, t)
		assert.Equal(t, http.StatusOK, response.Code)
	})
	t.Run("AccessControl allows viewing another org with exact permissions", func(t *testing.T) {
		setAccessControlPermissions(sc.acmock, []*accesscontrol.Permission{{Action: ActionOrgsRead, Scope: accesscontrol.Scope("orgs", "name", "TestOrg2")}})
		response := callAPI(sc.server, http.MethodGet, fmt.Sprintf(getOrgsByNameURL, "TestOrg2"), nil, t)
		assert.Equal(t, http.StatusOK, response.Code)
	})
	t.Run("AccessControl prevents viewing another org with too narrow permissions", func(t *testing.T) {
		setAccessControlPermissions(sc.acmock, []*accesscontrol.Permission{{Action: ActionOrgsRead, Scope: accesscontrol.Scope("orgs", "name", "TestOrg1")}})
		response := callAPI(sc.server, http.MethodGet, fmt.Sprintf(getOrgsByNameURL, "TestOrg2"), nil, t)
		assert.Equal(t, http.StatusForbidden, response.Code)
	})
	t.Run("AccessControl prevents viewing another org with incorrect permissions", func(t *testing.T) {
		setAccessControlPermissions(sc.acmock, []*accesscontrol.Permission{{Action: "orgs:invalid"}})
		response := callAPI(sc.server, http.MethodGet, fmt.Sprintf(getOrgsByNameURL, "TestOrg2"), nil, t)
		assert.Equal(t, http.StatusForbidden, response.Code)
	})
}

func TestAPIEndpoint_PutCurrentOrg_LegacyAccessControl(t *testing.T) {
	sc := setupHTTPServer(t, false)

	_, err := sc.db.CreateOrgWithMember("TestOrg", testUserID)
	require.NoError(t, err)

	input := strings.NewReader(testUpdateOrgNameForm)

	setInitCtxSignedInViewer(sc.initCtx)
	t.Run("Viewer cannot update current org", func(t *testing.T) {
		response := callAPI(sc.server, http.MethodPut, putCurrentOrgURL, input, t)
		assert.Equal(t, http.StatusForbidden, response.Code)
	})

	setInitCtxSignedInOrgAdmin(sc.initCtx)
	t.Run("Admin can update current org", func(t *testing.T) {
		response := callAPI(sc.server, http.MethodPut, putCurrentOrgURL, input, t)
		assert.Equal(t, http.StatusOK, response.Code)
	})
}

func TestAPIEndpoint_PutCurrentOrg_AccessControl(t *testing.T) {
	sc := setupHTTPServer(t, true)
	setInitCtxSignedInViewer(sc.initCtx)

	_, err := sc.db.CreateOrgWithMember("TestOrg", testUserID)
	require.NoError(t, err)

	input := strings.NewReader(testUpdateOrgNameForm)
	t.Run("AccessControl allows updating current org with correct permissions", func(t *testing.T) {
		setAccessControlPermissions(sc.acmock, []*accesscontrol.Permission{{Action: ActionOrgsWrite, Scope: ScopeOrgsAll}})
		response := callAPI(sc.server, http.MethodPut, putCurrentOrgURL, input, t)
		assert.Equal(t, http.StatusOK, response.Code)
	})

	input = strings.NewReader(testUpdateOrgNameForm)
	t.Run("AccessControl allows updating current org with exact permissions", func(t *testing.T) {
		setAccessControlPermissions(sc.acmock, []*accesscontrol.Permission{{Action: ActionOrgsWrite, Scope: accesscontrol.Scope("orgs", "id", "1")}})
		response := callAPI(sc.server, http.MethodPut, putCurrentOrgURL, input, t)
		assert.Equal(t, http.StatusOK, response.Code)
	})

	t.Run("AccessControl prevents updating current org with too narrow permissions", func(t *testing.T) {
		setAccessControlPermissions(sc.acmock, []*accesscontrol.Permission{{Action: ActionOrgsWrite, Scope: accesscontrol.Scope("orgs", "id", "2")}})
		response := callAPI(sc.server, http.MethodPut, putCurrentOrgURL, input, t)
		assert.Equal(t, http.StatusForbidden, response.Code)
	})

	t.Run("AccessControl prevents updating current org with incorrect permissions", func(t *testing.T) {
		setAccessControlPermissions(sc.acmock, []*accesscontrol.Permission{{Action: "orgs:invalid"}})
		response := callAPI(sc.server, http.MethodPut, putCurrentOrgURL, input, t)
		assert.Equal(t, http.StatusForbidden, response.Code)
	})
}

func TestAPIEndpoint_PutOrg_LegacyAccessControl(t *testing.T) {
	sc := setupHTTPServer(t, false)
	setInitCtxSignedInViewer(sc.initCtx)

	// Create two orgs, to update another one than the logged in one
	_, err := sc.db.CreateOrgWithMember("TestOrg", testUserID)
	require.NoError(t, err)
	_, err = sc.db.CreateOrgWithMember("TestOrg2", testUserID)
	require.NoError(t, err)

	input := strings.NewReader(testUpdateOrgNameForm)

	t.Run("Viewer cannot update another org", func(t *testing.T) {
		response := callAPI(sc.server, http.MethodPut, fmt.Sprintf(putOrgsURL, 2), input, t)
		assert.Equal(t, http.StatusForbidden, response.Code)
	})

	sc.initCtx.SignedInUser.IsGrafanaAdmin = true
	t.Run("Grafana Admin can update another org", func(t *testing.T) {
		response := callAPI(sc.server, http.MethodPut, fmt.Sprintf(putOrgsURL, 2), input, t)
		assert.Equal(t, http.StatusOK, response.Code)
	})
}

func TestAPIEndpoint_PutOrg_AccessControl(t *testing.T) {
	sc := setupHTTPServer(t, true)
	setInitCtxSignedInViewer(sc.initCtx)

	// Create two orgs, to update another one than the logged in one
	_, err := sc.db.CreateOrgWithMember("TestOrg", testUserID)
	require.NoError(t, err)
	_, err = sc.db.CreateOrgWithMember("TestOrg2", testUserID)
	require.NoError(t, err)

	input := strings.NewReader(testUpdateOrgNameForm)
	t.Run("AccessControl allows updating another org with correct permissions", func(t *testing.T) {
		setAccessControlPermissions(sc.acmock, []*accesscontrol.Permission{{Action: ActionOrgsWrite, Scope: ScopeOrgsAll}})
		response := callAPI(sc.server, http.MethodPut, fmt.Sprintf(putOrgsURL, 2), input, t)
		assert.Equal(t, http.StatusOK, response.Code)
	})

	input = strings.NewReader(testUpdateOrgNameForm)
	t.Run("AccessControl allows updating another org with exact permissions", func(t *testing.T) {
		setAccessControlPermissions(sc.acmock, []*accesscontrol.Permission{{Action: ActionOrgsWrite, Scope: accesscontrol.Scope("orgs", "id", "2")}})
		response := callAPI(sc.server, http.MethodPut, fmt.Sprintf(putOrgsURL, 2), input, t)
		assert.Equal(t, http.StatusOK, response.Code)
	})

	input = strings.NewReader(testUpdateOrgNameForm)
	t.Run("AccessControl prevents updating another org with too narrow permissions", func(t *testing.T) {
		setAccessControlPermissions(sc.acmock, []*accesscontrol.Permission{{Action: ActionOrgsWrite, Scope: accesscontrol.Scope("orgs", "id", "1")}})
		response := callAPI(sc.server, http.MethodPut, fmt.Sprintf(putOrgsURL, 2), input, t)
		assert.Equal(t, http.StatusForbidden, response.Code)
	})

	t.Run("AccessControl prevents updating another org with incorrect permissions", func(t *testing.T) {
		setAccessControlPermissions(sc.acmock, []*accesscontrol.Permission{{Action: "orgs:invalid"}})
		response := callAPI(sc.server, http.MethodPut, fmt.Sprintf(putOrgsURL, 2), input, t)
		assert.Equal(t, http.StatusForbidden, response.Code)
	})
}

func TestAPIEndpoint_PutCurrentOrgAddress_LegacyAccessControl(t *testing.T) {
	sc := setupHTTPServer(t, false)

	_, err := sc.db.CreateOrgWithMember("TestOrg", testUserID)
	require.NoError(t, err)

	input := strings.NewReader(testUpdateOrgAddressForm)

	setInitCtxSignedInViewer(sc.initCtx)
	t.Run("Viewer cannot update current org address", func(t *testing.T) {
		response := callAPI(sc.server, http.MethodPut, putCurrentOrgAddressURL, input, t)
		assert.Equal(t, http.StatusForbidden, response.Code)
	})

	setInitCtxSignedInOrgAdmin(sc.initCtx)
	t.Run("Admin can update current org address", func(t *testing.T) {
		response := callAPI(sc.server, http.MethodPut, putCurrentOrgAddressURL, input, t)
		assert.Equal(t, http.StatusOK, response.Code)
	})
}

func TestAPIEndpoint_PutCurrentOrgAddress_AccessControl(t *testing.T) {
	sc := setupHTTPServer(t, true)
	setInitCtxSignedInViewer(sc.initCtx)

	_, err := sc.db.CreateOrgWithMember("TestOrg", testUserID)
	require.NoError(t, err)

	input := strings.NewReader(testUpdateOrgAddressForm)
	t.Run("AccessControl allows updating current org address with correct permissions", func(t *testing.T) {
		setAccessControlPermissions(sc.acmock, []*accesscontrol.Permission{{Action: ActionOrgsWrite, Scope: ScopeOrgsAll}})
		response := callAPI(sc.server, http.MethodPut, putCurrentOrgAddressURL, input, t)
		assert.Equal(t, http.StatusOK, response.Code)
	})

	input = strings.NewReader(testUpdateOrgAddressForm)
	t.Run("AccessControl allows updating current org address with exact permissions", func(t *testing.T) {
		setAccessControlPermissions(sc.acmock, []*accesscontrol.Permission{{Action: ActionOrgsWrite, Scope: accesscontrol.Scope("orgs", "id", "1")}})
		response := callAPI(sc.server, http.MethodPut, putCurrentOrgAddressURL, input, t)
		assert.Equal(t, http.StatusOK, response.Code)
	})

	t.Run("AccessControl prevents updating current org address with incorrect permissions", func(t *testing.T) {
		setAccessControlPermissions(sc.acmock, []*accesscontrol.Permission{{Action: "orgs:invalid"}})
		response := callAPI(sc.server, http.MethodPut, putCurrentOrgAddressURL, input, t)
		assert.Equal(t, http.StatusForbidden, response.Code)
	})
}

func TestAPIEndpoint_PutOrgAddress_LegacyAccessControl(t *testing.T) {
	sc := setupHTTPServer(t, false)
	setInitCtxSignedInViewer(sc.initCtx)

	// Create two orgs, to update another one than the logged in one
	_, err := sc.db.CreateOrgWithMember("TestOrg", testUserID)
	require.NoError(t, err)
	_, err = sc.db.CreateOrgWithMember("TestOrg2", testUserID)
	require.NoError(t, err)

	input := strings.NewReader(testUpdateOrgAddressForm)

	t.Run("Viewer cannot update another org address", func(t *testing.T) {
		response := callAPI(sc.server, http.MethodPut, fmt.Sprintf(putOrgsAddressURL, 2), input, t)
		assert.Equal(t, http.StatusForbidden, response.Code)
	})

	sc.initCtx.SignedInUser.IsGrafanaAdmin = true
	t.Run("Grafana Admin can update another org address", func(t *testing.T) {
		response := callAPI(sc.server, http.MethodPut, fmt.Sprintf(putOrgsAddressURL, 2), input, t)
		assert.Equal(t, http.StatusOK, response.Code)
	})
}

func TestAPIEndpoint_PutOrgAddress_AccessControl(t *testing.T) {
	sc := setupHTTPServer(t, true)
	setInitCtxSignedInViewer(sc.initCtx)

	// Create two orgs, to update another one than the logged in one
	_, err := sc.db.CreateOrgWithMember("TestOrg", testUserID)
	require.NoError(t, err)
	_, err = sc.db.CreateOrgWithMember("TestOrg2", testUserID)
	require.NoError(t, err)

	input := strings.NewReader(testUpdateOrgAddressForm)
	t.Run("AccessControl allows updating another org address with correct permissions", func(t *testing.T) {
		setAccessControlPermissions(sc.acmock, []*accesscontrol.Permission{{Action: ActionOrgsWrite, Scope: ScopeOrgsAll}})
		response := callAPI(sc.server, http.MethodPut, fmt.Sprintf(putOrgsAddressURL, 2), input, t)
		assert.Equal(t, http.StatusOK, response.Code)
	})

	input = strings.NewReader(testUpdateOrgAddressForm)
	t.Run("AccessControl prevents updating another org address with too narrow permissions", func(t *testing.T) {
		setAccessControlPermissions(sc.acmock, []*accesscontrol.Permission{{Action: ActionOrgsWrite, Scope: accesscontrol.Scope("orgs", "id", "1")}})
		response := callAPI(sc.server, http.MethodPut, fmt.Sprintf(putOrgsAddressURL, 2), input, t)
		assert.Equal(t, http.StatusForbidden, response.Code)
	})

	t.Run("AccessControl prevents updating another org address with incorrect permissions", func(t *testing.T) {
		setAccessControlPermissions(sc.acmock, []*accesscontrol.Permission{{Action: "orgs:invalid"}})
		response := callAPI(sc.server, http.MethodPut, fmt.Sprintf(putOrgsAddressURL, 2), input, t)
		assert.Equal(t, http.StatusForbidden, response.Code)
	})
}
