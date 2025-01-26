import { NgModule } from '@angular/core';
import {
  CoreModule,
  ViewContext,
  FormsModule,
  gettext,
  hookRoute,
} from '@c8y/ngx-components';
import { ContainerManagementComponent } from './container-management.component';
import { BsDropdownModule } from 'ngx-bootstrap/dropdown';
import { ContainerFilterPipe } from './container-filter.pipe';
import { ContainerGuard } from '../../src/shared/container.guard';

@NgModule({
  declarations: [ContainerManagementComponent, ContainerFilterPipe],
  imports: [CoreModule, FormsModule, BsDropdownModule],
  providers: [
    hookRoute({
      path: 'containers',
      context: ViewContext.Device,
      component: ContainerManagementComponent,
      label: gettext('Containers'),
      priority: 999,
      icon: 'package',
      canActivate: [ContainerGuard],
    }),
  ],
})
export class ContainerManagementModule {}
