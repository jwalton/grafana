import React, { PureComponent } from 'react';
import { css } from '@emotion/css';
import { Select, Alert, Label, stylesFactory } from '@grafana/ui';
import {
  LiveChannelScope,
  LiveChannelAddress,
  LiveChannelSupport,
  SelectableValue,
  StandardEditorProps,
  GrafanaTheme,
} from '@grafana/data';

import { LivePanelOptions } from './types';
import { getGrafanaLiveScopes } from 'app/features/live';
import { config } from 'app/core/config';

type Props = StandardEditorProps<LiveChannelAddress, any, LivePanelOptions>;

const scopes: Array<SelectableValue<LiveChannelScope>> = [
  { label: 'Grafana', value: LiveChannelScope.Grafana, description: 'Core grafana live features' },
  { label: 'Data Sources', value: LiveChannelScope.DataSource, description: 'Data sources with live support' },
  { label: 'Plugins', value: LiveChannelScope.Plugin, description: 'Plugins with live support' },
];

interface State {
  namespaces: Array<SelectableValue<string>>;
  paths: Array<SelectableValue<string>>;
  support?: LiveChannelSupport;
}

export class LiveChannelEditor extends PureComponent<Props, State> {
  state: State = {
    namespaces: [],
    paths: [],
  };

  async componentDidMount() {
    this.updateSelectOptions();
  }

  async componentDidUpdate(oldProps: Props) {
    if (this.props.value !== oldProps.value) {
      this.updateSelectOptions();
    }
  }

  async getScopeDetails() {
    const { scope, namespace } = this.props.value;
    const srv = getGrafanaLiveScopes();

    if (!srv.doesScopeExist(scope)) {
      return {
        namespaces: [],
        support: undefined,
      };
    }

    const namespaces = await srv.getNamespaces(scope);
    const support = namespace ? await srv.getChannelSupport(scope, namespace) : undefined;
    return {
      namespaces,
      support,
    };
  }

  async updateSelectOptions() {
    const { namespaces, support } = await this.getScopeDetails();

    this.setState({
      namespaces,
      support,
      paths: [],
    });
  }

  onScopeChanged = (v: SelectableValue<LiveChannelScope>) => {
    if (v.value) {
      this.props.onChange({
        scope: v.value,
        namespace: (undefined as unknown) as string,
        path: (undefined as unknown) as string,
      } as LiveChannelAddress);
    }
  };

  onNamespaceChanged = (v: SelectableValue<string>) => {
    const update = {
      scope: this.props.value?.scope,
      path: (undefined as unknown) as string,
    } as LiveChannelAddress;

    if (v.value) {
      update.namespace = v.value;
    }
    this.props.onChange(update);
  };

  onPathChanged = (v: SelectableValue<string>) => {
    const { value, onChange } = this.props;
    const update = {
      scope: value.scope,
      namespace: value.namespace,
    } as LiveChannelAddress;
    if (v.value) {
      update.path = v.value;
    }
    onChange(update);
  };

  render() {
    const { namespaces, paths } = this.state;
    const { scope, namespace, path } = this.props.value;
    const style = getStyles(config.theme);

    return (
      <>
        <Alert title="Grafana Live" severity="info">
          This supports real-time event streams in grafana core. This feature is under heavy development. Expect the
          intefaces and structures to change as this becomes more production ready.
        </Alert>

        <div>
          <div className={style.dropWrap}>
            <Label>Scope</Label>
            <Select
              menuShouldPortal
              options={scopes}
              value={scopes.find((s) => s.value === scope)}
              onChange={this.onScopeChanged}
            />
          </div>

          {scope && (
            <div className={style.dropWrap}>
              <Label>Namespace</Label>
              <Select
                menuShouldPortal
                options={namespaces}
                value={
                  namespaces.find((s) => s.value === namespace) ??
                  (namespace ? { label: namespace, value: namespace } : undefined)
                }
                onChange={this.onNamespaceChanged}
                allowCustomValue={true}
                backspaceRemovesValue={true}
              />
            </div>
          )}

          {scope && namespace && (
            <div className={style.dropWrap}>
              <Label>Path</Label>
              <Select
                menuShouldPortal
                options={paths}
                value={findPathOption(paths, path)}
                onChange={this.onPathChanged}
                allowCustomValue={true}
                backspaceRemovesValue={true}
              />
            </div>
          )}
        </div>
      </>
    );
  }
}

function findPathOption(paths: Array<SelectableValue<string>>, path?: string): SelectableValue<string> | undefined {
  const v = paths.find((s) => s.value === path);
  if (v) {
    return v;
  }
  if (path) {
    return { label: path, value: path };
  }
  return undefined;
}

const getStyles = stylesFactory((theme: GrafanaTheme) => ({
  dropWrap: css`
    margin-bottom: ${theme.spacing.sm};
  `,
}));
