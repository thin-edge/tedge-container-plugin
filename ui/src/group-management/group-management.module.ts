import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import {
  CoreModule,
  HOOK_TABS,
  ViewContext,
  FormsModule,
  TabsModule,
  HOOK_ROUTE,
  Route,
} from '@c8y/ngx-components';
import { BsDropdownModule } from 'ngx-bootstrap/dropdown';
import { GroupManagementComponent } from './group-management.component';
import { ContainerListItemComponent } from './container-list-item/container-list-item.component';
import { GroupFilterPipe } from './group-filter.pipe';
import { ContainerGuard } from '../shared/container.guard';

const tabHook = {
  provide: HOOK_ROUTE,
  useValue: [
    {
      path: 'ContainerGroups',
      context: ViewContext.Device,
      component: GroupManagementComponent,
      label: 'Container Groups',
      priority: 998,
      icon: 'packages',
      canActivate: [ContainerGuard],
    },
  ],
  multi: true,
};

@NgModule({
  declarations: [
    GroupManagementComponent,
    ContainerListItemComponent,
    GroupFilterPipe,
  ],
  imports: [CoreModule, FormsModule, BsDropdownModule],
  entryComponents: [GroupManagementComponent],
  providers: [tabHook],
})
export class GroupManagementModule {}
