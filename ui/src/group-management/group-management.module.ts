import { NgModule } from '@angular/core';
import {
  CoreModule,
  ViewContext,
  FormsModule,
  gettext,
  hookRoute,
} from '@c8y/ngx-components';
import { BsDropdownModule } from 'ngx-bootstrap/dropdown';
import { GroupManagementComponent } from './group-management.component';
import { ContainerListItemComponent } from './container-list-item/container-list-item.component';
import { GroupFilterPipe } from './group-filter.pipe';
import { ContainerGuard } from '../shared/container.guard';

@NgModule({
  declarations: [
    GroupManagementComponent,
    ContainerListItemComponent,
    GroupFilterPipe,
  ],
  imports: [CoreModule, FormsModule, BsDropdownModule],
  providers: [
    hookRoute({
      path: 'ContainerGroups',
      context: ViewContext.Device,
      component: GroupManagementComponent,
      label: gettext('Container Groups'),
      priority: 998,
      icon: 'packages',
      canActivate: [ContainerGuard],
    }),
  ],
})
export class GroupManagementModule {}
