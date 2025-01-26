import { NgModule } from '@angular/core';
import {
  CoreModule,
  ViewContext,
  FormsModule,
  hookRoute,
  gettext,
} from '@c8y/ngx-components';
import { ContainerInfoComponent } from './container-info.component';
import { ContainerInfoGuard } from './container-info.guard';

@NgModule({
  declarations: [ContainerInfoComponent],
  imports: [CoreModule, FormsModule],
  providers: [
    hookRoute({
      path: `Info`,
      label: gettext('Info'),
      context: ViewContext.Service,
      component: ContainerInfoComponent,
      priority: 1000,
      icon: 'asterisk',
      canActivate: [ContainerInfoGuard],
    }),
  ],
})
export class ContainerInfoModule {}
