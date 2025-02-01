package cli

import (
	"context"

	"github.com/google/uuid"
	"github.com/replicatedhq/embedded-cluster/pkg/metrics"
	preflightstypes "github.com/replicatedhq/embedded-cluster/pkg/preflights/types"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
)

type InstallReporter struct {
	license   *kotsv1beta1.License
	clusterID uuid.UUID
	cmd       string
}

func NewInstallReporter(license *kotsv1beta1.License, clusterID uuid.UUID, cmd string) *InstallReporter {
	return &InstallReporter{
		license:   license,
		clusterID: clusterID,
		cmd:       cmd,
	}
}

func (r *InstallReporter) ReportInstallationStarted(ctx context.Context) {
	metrics.ReportInstallationStarted(ctx, r.license, r.clusterID)
}

func (r *InstallReporter) ReportInstallationSucceeded(ctx context.Context) {
	metrics.ReportInstallationSucceeded(ctx, r.license, r.clusterID)
}

func (r *InstallReporter) ReportInstallationFailed(ctx context.Context, err error) {
	metrics.ReportInstallationFailed(ctx, r.license, r.clusterID, err)
}

func (r *InstallReporter) ReportPreflightsFailed(ctx context.Context, output preflightstypes.Output, bypassed bool) {
	metrics.ReportPreflightsFailed(ctx, metrics.BaseURL(r.license), r.clusterID, output, bypassed, r.cmd)
}

type JoinReporter struct {
	baseURL   string
	clusterID uuid.UUID
	cmd       string
}

func NewJoinReporter(baseURL string, clusterID uuid.UUID, cmd string) *JoinReporter {
	return &JoinReporter{
		baseURL:   baseURL,
		clusterID: clusterID,
		cmd:       cmd,
	}
}

func (r *JoinReporter) ReportJoinStarted(ctx context.Context) {
	metrics.ReportJoinStarted(ctx, r.baseURL, r.clusterID)
}

func (r *JoinReporter) ReportJoinSucceeded(ctx context.Context) {
	metrics.ReportJoinSucceeded(ctx, r.baseURL, r.clusterID)
}

func (r *JoinReporter) ReportJoinFailed(ctx context.Context, err error) {
	metrics.ReportJoinFailed(ctx, r.baseURL, r.clusterID, err)
}

func (r *JoinReporter) ReportPreflightsFailed(ctx context.Context, output preflightstypes.Output, bypassed bool) {
	metrics.ReportPreflightsFailed(ctx, r.baseURL, r.clusterID, output, bypassed, r.cmd)
}
