import { NgModule } from '@angular/core';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import {
  BootstrapComponent,
  CoreModule,
  RouterModule,
} from '@c8y/ngx-components';

@NgModule({
  imports: [
    BrowserAnimationsModule,
    RouterModule.forRoot([]),
    CoreModule.forRoot(),
  ],
  bootstrap: [BootstrapComponent],
})
export class AppModule {}
