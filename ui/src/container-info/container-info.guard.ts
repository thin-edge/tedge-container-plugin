import { Injectable } from '@angular/core';
import { CanActivate, ActivatedRouteSnapshot } from '@angular/router';
import { Observable } from 'rxjs';

@Injectable({ providedIn: 'root' })
export class ContainerInfoGuard implements CanActivate {
  canActivate(
    route: ActivatedRouteSnapshot
  ): Observable<boolean> | Promise<boolean> | boolean {
    const contextData = route.data.contextData || route.parent.data.contextData;
    return contextData['serviceType'] == 'container' || contextData['serviceType'] == 'container-group';
  }
}
