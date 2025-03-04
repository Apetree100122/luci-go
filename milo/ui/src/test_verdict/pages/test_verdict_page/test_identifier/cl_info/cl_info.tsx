// Copyright 2023 The LUCI Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import styled from '@emotion/styled';
import Divider from '@mui/material/Divider';
import Grid from '@mui/material/Grid';
import Link from '@mui/material/Link';
import Tooltip, { TooltipProps, tooltipClasses } from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';

import {
  getBuildURLPathFromBuildId,
  getGerritChangeURL,
} from '@/common/tools/url_utils';
import { GerritChange } from '@/proto/go.chromium.org/luci/resultdb/proto/v1/common.pb';

import { useInvocationID, useSources } from '../../context';

const HtmlTooltip = styled(({ className, ...props }: TooltipProps) => (
  <Tooltip {...props} classes={{ popper: className }} />
))(() => ({
  [`& .${tooltipClasses.arrow}`]: {
    color: '#f5f5f9',
    '&::before': {
      backgroundColor: 'white',
      border: '1px solid #dadde9',
    },
  },
  [`& .${tooltipClasses.tooltip}`]: {
    backgroundColor: 'white',
    color: 'rgba(0, 0, 0, 0.87)',
    maxWidth: 220,
    fontSize: '0.9rem',
    border: '1px solid #dadde9',
    boxShadow: '0px 5px 8px -3px rgba(0,0,0,0.73)',
  },
}));

interface ChangelistLinkProps {
  changelist: GerritChange;
}

export function ChangelistLink({ changelist }: ChangelistLinkProps) {
  return (
    <Link target="_blank" href={getGerritChangeURL(changelist)}>
      {changelist.change}/{changelist.patchset}
    </Link>
  );
}

export function CLInfo() {
  const sources = useSources();
  const invID = useInvocationID();
  const buildID = invID.split('-')[1];
  return (
    <>
      <Grid
        item
        container
        columnGap={1}
        sx={{
          px: 0.5,
        }}
      >
        {sources && sources.changelists && (
          <>
            CL: <ChangelistLink changelist={sources.changelists[0]} />
            {sources.changelists.length > 1 && (
              <HtmlTooltip
                arrow
                placement="bottom"
                sx={{
                  textDecoration: 'underline',
                }}
                title={
                  <Grid container rowGap={1} padding={1}>
                    {sources.changelists.map((changelist, i) => (
                      <Grid item key={changelist.change}>
                        {i > 0 && (
                          <li>
                            <ChangelistLink changelist={changelist} />
                          </li>
                        )}
                      </Grid>
                    ))}
                  </Grid>
                }
              >
                <Typography
                  sx={{
                    fontSize: '14px',
                    display: 'inline',
                    lineHeight: 'normal',
                    padding: 0,
                    textDecoration: 'underline',
                    textDecorationThickness: 'auto',
                    cursor: 'help',
                  }}
                >
                  and {sources.changelists.length - 1} more.
                </Typography>
              </HtmlTooltip>
            )}
            <Divider orientation="vertical" flexItem />
          </>
        )}
        Build:
        <Link target="_blank" href={getBuildURLPathFromBuildId(buildID)}>
          {buildID}
        </Link>
      </Grid>
    </>
  );
}
