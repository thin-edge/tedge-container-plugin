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
import { ContainerManagementComponent } from './container-management.component';
import { BsDropdownModule } from 'ngx-bootstrap/dropdown';
import { ContainerFilterPipe } from './container-filter.pipe';
import { ContainerGuard } from '../../src/shared/container.guard';

const tabHook = {
  provide: HOOK_ROUTE,
  useValue: [
    {
      path: 'containers',
      context: ViewContext.Device,
      component: ContainerManagementComponent,
      label: 'Containers',
      priority: 999,
      icon: 'package',
      canActivate: [ContainerGuard],
    },
  ],
  multi: true,
};

@NgModule({
  declarations: [ContainerManagementComponent, ContainerFilterPipe],
  imports: [CoreModule, FormsModule, BsDropdownModule],
  entryComponents: [ContainerManagementComponent],
  providers: [tabHook],
})
export class ContainerManagementModule {}
