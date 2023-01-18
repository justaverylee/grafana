package dashboards

import (
	"context"
	"io"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/grafana/grafana/pkg/plugins"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDashboardFileStore(t *testing.T) {
	m := setupPluginDashboardsForTest(t)

	t.Run("Input validation", func(t *testing.T) {
		t.Run("ListPluginDashboardFiles", func(t *testing.T) {
			testCases := []struct {
				name string
				args *ListPluginDashboardFilesArgs
			}{
				{
					name: "nil args should return error",
				},
				{
					name: "empty args.PluginID should return error",
					args: &ListPluginDashboardFilesArgs{},
				},
				{
					name: "args.PluginID with only space should return error",
					args: &ListPluginDashboardFilesArgs{PluginID: " \t "},
				},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					res, err := m.ListPluginDashboardFiles(context.Background(), tc.args)
					assert.Error(t, err)
					assert.Nil(t, res)
				})
			}
		})

		t.Run("GetPluginDashboardFileContents", func(t *testing.T) {
			testCases := []struct {
				name string
				args *GetPluginDashboardFileContentsArgs
			}{
				{
					name: "nil args should return error",
				},
				{
					name: "empty args.PluginID should return error",
					args: &GetPluginDashboardFileContentsArgs{},
				},
				{
					name: "args.PluginID with only space should return error",
					args: &GetPluginDashboardFileContentsArgs{PluginID: " "},
				},
				{
					name: "empty args.FileReference should return error",
					args: &GetPluginDashboardFileContentsArgs{
						PluginID: "pluginWithDashboards",
					},
				},
				{
					name: "args.FileReference with only space should return error",
					args: &GetPluginDashboardFileContentsArgs{
						PluginID:      "pluginWithDashboard",
						FileReference: " \t",
					},
				},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					res, err := m.GetPluginDashboardFileContents(context.Background(), tc.args)
					assert.Error(t, err)
					assert.Nil(t, res)
				})
			}
		})
	})

	t.Run("Plugin without dashboards", func(t *testing.T) {
		t.Run("Should return zero file references", func(t *testing.T) {
			res, err := m.ListPluginDashboardFiles(context.Background(), &ListPluginDashboardFilesArgs{
				PluginID: "pluginWithoutDashboards",
			})
			require.NoError(t, err)
			require.NotNil(t, res)
			require.Len(t, res.FileReferences, 0)
		})

		t.Run("Should return file not found error when trying to get non-existing plugin dashboard file content", func(t *testing.T) {
			res, err := m.GetPluginDashboardFileContents(context.Background(), &GetPluginDashboardFileContentsArgs{
				PluginID:      "pluginWithoutDashboards",
				FileReference: "dashboards/dash2.json",
			})
			require.Error(t, err)
			require.EqualError(t, err, "plugin dashboard file not found")
			require.Nil(t, res)
		})
	})

	t.Run("Plugin with dashboards", func(t *testing.T) {
		t.Run("Should return two file references", func(t *testing.T) {
			res, err := m.ListPluginDashboardFiles(context.Background(), &ListPluginDashboardFilesArgs{
				PluginID: "pluginWithDashboards",
			})
			require.NoError(t, err)
			require.NotNil(t, res)
			require.Len(t, res.FileReferences, 2)
		})

		t.Run("With filesystem", func(t *testing.T) {
			origOpenDashboardFile := openDashboardFile
			mapFs := fstest.MapFS{
				"dashboards/dash1.json": {
					Data: []byte("dash1"),
				},
				"dashboards/dash2.json": {
					Data: []byte("dash2"),
				},
				"dashboards/dash3.json": {
					Data: []byte("dash3"),
				},
				"dash2.json": {
					Data: []byte("dash2"),
				},
			}
			openDashboardFile = func(p plugins.PluginDTO, name string) (fs.File, error) {
				f, err := mapFs.Open(name)
				require.NoError(t, err)
				return f, nil
			}
			t.Cleanup(func() {
				openDashboardFile = origOpenDashboardFile
			})

			t.Run("Should return file not found error when trying to get non-existing plugin dashboard file content", func(t *testing.T) {
				res, err := m.GetPluginDashboardFileContents(context.Background(), &GetPluginDashboardFileContentsArgs{
					PluginID:      "pluginWithDashboards",
					FileReference: "dashboards/dash3.json",
				})
				require.Error(t, err)
				require.EqualError(t, err, "plugin dashboard file not found")
				require.Nil(t, res)
			})

			t.Run("Should return file content for dashboards/dash1.json", func(t *testing.T) {
				res, err := m.GetPluginDashboardFileContents(context.Background(), &GetPluginDashboardFileContentsArgs{
					PluginID:      "pluginWithDashboards",
					FileReference: "dashboards/dash1.json",
				})
				require.NoError(t, err)
				require.NotNil(t, res)
				require.NotNil(t, res.Content)
				b, err := io.ReadAll(res.Content)
				require.NoError(t, err)
				require.Equal(t, "dash1", string(b))
			})

			t.Run("Should return file content for dashboards/dash2.json", func(t *testing.T) {
				res, err := m.GetPluginDashboardFileContents(context.Background(), &GetPluginDashboardFileContentsArgs{
					PluginID:      "pluginWithDashboards",
					FileReference: "dashboards/dash2.json",
				})
				require.NoError(t, err)
				require.NotNil(t, res)
				require.NotNil(t, res.Content)
				b, err := io.ReadAll(res.Content)
				require.NoError(t, err)
				require.Equal(t, "dash2", string(b))
			})

			t.Run("Should return error when trying to read relative file", func(t *testing.T) {
				res, err := m.GetPluginDashboardFileContents(context.Background(), &GetPluginDashboardFileContentsArgs{
					PluginID:      "pluginWithDashboards",
					FileReference: "dashboards/../dash2.json",
				})
				require.Error(t, err)
				require.EqualError(t, err, "plugin dashboard file not found")
				require.Nil(t, res)
			})
		})
	})
}

func setupPluginDashboardsForTest(t *testing.T) *FileStoreManager {
	t.Helper()

	p1 := &plugins.Plugin{
		JSONData: plugins.JSONData{
			ID: "pluginWithoutDashboards",
			Includes: []*plugins.Includes{
				{
					Type: "page",
				},
			},
		},
	}

	p2 := &plugins.Plugin{
		JSONData: plugins.JSONData{
			ID: "pluginWithDashboards",
			Includes: []*plugins.Includes{
				{
					Type: "page",
				},
				{
					Type: "dashboard",
					Path: "dashboards/dash1.json",
				},
				{
					Type: "dashboard",
					Path: "dashboards/dash2.json",
				},
			},
		},
	}

	return &FileStoreManager{
		pluginStore: &plugins.FakePluginStore{
			PluginList: []plugins.PluginDTO{p1.ToDTO(), p2.ToDTO()},
		},
	}
}
