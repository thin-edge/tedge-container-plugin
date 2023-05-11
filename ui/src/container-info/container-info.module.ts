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
import { ContainerInfoComponent } from './container-info.component';
import { ContainerInfoGuard } from './container-info.guard';

const tabHook = {
  provide: HOOK_ROUTE,
  useValue: [
    {
      path: 'Info',
      context: ViewContext.Service,
      component: ContainerInfoComponent,
      label: 'Info',
      priority: 1000,
      icon: 'asterisk',
      canActivate: [ContainerInfoGuard],
    },
  ],
  multi: true,
};

@NgModule({
  declarations: [ContainerInfoComponent],
  imports: [CoreModule, FormsModule],
  entryComponents: [ContainerInfoComponent],
  providers: [tabHook],
})
export class ContainerInfoModule {}
