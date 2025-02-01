package migratev2

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	ecv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Test_setV2MigrationInProgress(t *testing.T) {
	// Discard log messages
	old := slog.SetLogLoggerLevel(slog.LevelError)
	defer slog.SetLogLoggerLevel(old)

	scheme := runtime.NewScheme()
	require.NoError(t, ecv1beta1.AddToScheme(scheme))

	type args struct {
		installation *ecv1beta1.Installation
	}
	tests := []struct {
		name        string
		args        args
		expectError bool
	}{
		{
			name: "set v2 migration in progress",
			args: args{
				installation: &ecv1beta1.Installation{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-installation",
					},
					TypeMeta: metav1.TypeMeta{
						APIVersion: "embeddedcluster.replicated.com/v1beta1",
						Kind:       "Installation",
					},
				},
			},
			expectError: false,
		},
		{
			name: "updates the condition",
			args: args{
				installation: &ecv1beta1.Installation{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-installation",
					},
					TypeMeta: metav1.TypeMeta{
						APIVersion: "embeddedcluster.replicated.com/v1beta1",
						Kind:       "Installation",
					},
					Status: ecv1beta1.InstallationStatus{
						Conditions: []metav1.Condition{
							{
								Type:    ecv1beta1.ConditionTypeV2MigrationInProgress,
								Status:  metav1.ConditionFalse,
								Reason:  "MigrationFailed",
								Message: "Migration failed",
							},
						},
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.args.installation).
				WithStatusSubresource(&ecv1beta1.Installation{}).
				Build()

			err := setV2MigrationInProgress(context.Background(), cli, tt.args.installation)
			require.NoError(t, err)

			// Verify that the condition was set correctly
			var updatedInstallation ecv1beta1.Installation
			err = cli.Get(context.Background(), client.ObjectKey{Name: "test-installation"}, &updatedInstallation)
			require.NoError(t, err)

			condition := meta.FindStatusCondition(updatedInstallation.Status.Conditions, ecv1beta1.ConditionTypeV2MigrationInProgress)
			require.NotNil(t, condition, "Expected V2MigrationInProgress condition to be set")
			require.Equal(t, metav1.ConditionTrue, condition.Status)
			require.Equal(t, "MigrationInProgress", condition.Reason)
			require.Empty(t, condition.Message)
		})
	}
}

func Test_setV2MigrationComplete(t *testing.T) {
	// Discard log messages
	old := slog.SetLogLoggerLevel(slog.LevelError)
	defer slog.SetLogLoggerLevel(old)

	scheme := runtime.NewScheme()
	require.NoError(t, ecv1beta1.AddToScheme(scheme))

	type args struct {
		installation *ecv1beta1.Installation
	}
	tests := []struct {
		name        string
		args        args
		expectError bool
	}{
		{
			name: "set v2 migration in progress",
			args: args{
				installation: &ecv1beta1.Installation{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-installation",
					},
					TypeMeta: metav1.TypeMeta{
						APIVersion: "embeddedcluster.replicated.com/v1beta1",
						Kind:       "Installation",
					},
				},
			},
			expectError: false,
		},
		{
			name: "updates the condition",
			args: args{
				installation: &ecv1beta1.Installation{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-installation",
					},
					TypeMeta: metav1.TypeMeta{
						APIVersion: "embeddedcluster.replicated.com/v1beta1",
						Kind:       "Installation",
					},
					Status: ecv1beta1.InstallationStatus{
						Conditions: []metav1.Condition{
							{
								Type:   ecv1beta1.ConditionTypeV2MigrationInProgress,
								Status: metav1.ConditionTrue,
								Reason: "MigrationInProgress",
							},
						},
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.args.installation).
				WithStatusSubresource(&ecv1beta1.Installation{}).
				Build()
			err := setV2MigrationComplete(context.Background(), cli, tt.args.installation)
			require.NoError(t, err)

			// Verify that the condition was set correctly
			var updatedInstallation ecv1beta1.Installation
			err = cli.Get(context.Background(), client.ObjectKey{Name: "test-installation"}, &updatedInstallation)
			require.NoError(t, err)

			condition := meta.FindStatusCondition(updatedInstallation.Status.Conditions, ecv1beta1.ConditionTypeV2MigrationInProgress)
			require.NotNil(t, condition, "Expected V2MigrationInProgress condition to be set")
			require.Equal(t, metav1.ConditionFalse, condition.Status)
			require.Equal(t, "MigrationComplete", condition.Reason)
			require.Empty(t, condition.Message)
		})
	}
}

func Test_setV2MigrationFailed(t *testing.T) {
	// Discard log messages
	old := slog.SetLogLoggerLevel(slog.LevelError)
	defer slog.SetLogLoggerLevel(old)

	scheme := runtime.NewScheme()
	require.NoError(t, ecv1beta1.AddToScheme(scheme))

	type args struct {
		installation *ecv1beta1.Installation
		failure      error
	}
	tests := []struct {
		name        string
		args        args
		expectError bool
	}{
		{
			name: "set v2 migration in progress",
			args: args{
				installation: &ecv1beta1.Installation{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-installation",
					},
					TypeMeta: metav1.TypeMeta{
						APIVersion: "embeddedcluster.replicated.com/v1beta1",
						Kind:       "Installation",
					},
				},
				failure: fmt.Errorf("failed migration"),
			},
			expectError: false,
		},
		{
			name: "updates the condition",
			args: args{
				installation: &ecv1beta1.Installation{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-installation",
					},
					TypeMeta: metav1.TypeMeta{
						APIVersion: "embeddedcluster.replicated.com/v1beta1",
						Kind:       "Installation",
					},
					Status: ecv1beta1.InstallationStatus{
						Conditions: []metav1.Condition{
							{
								Type:   ecv1beta1.ConditionTypeV2MigrationInProgress,
								Status: metav1.ConditionTrue,
								Reason: "MigrationInProgress",
							},
						},
					},
				},
				failure: fmt.Errorf("failed migration"),
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.args.installation).
				WithStatusSubresource(&ecv1beta1.Installation{}).
				Build()

			err := setV2MigrationFailed(context.Background(), cli, tt.args.installation, tt.args.failure)
			require.NoError(t, err)

			// Verify that the condition was set correctly
			var updatedInstallation ecv1beta1.Installation
			err = cli.Get(context.Background(), client.ObjectKey{Name: "test-installation"}, &updatedInstallation)
			require.NoError(t, err)

			condition := meta.FindStatusCondition(updatedInstallation.Status.Conditions, ecv1beta1.ConditionTypeV2MigrationInProgress)
			require.NotNil(t, condition, "Expected V2MigrationInProgress condition to be set")
			require.Equal(t, metav1.ConditionFalse, condition.Status)
			require.Equal(t, "MigrationFailed", condition.Reason)
			require.Equal(t, condition.Message, tt.args.failure.Error())
		})
	}
}
