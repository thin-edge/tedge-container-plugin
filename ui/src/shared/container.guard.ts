import { Injectable } from '@angular/core';
import { ActivatedRouteSnapshot, CanActivate } from '@angular/router';
import { InventoryService } from '@c8y/client';
import { get } from 'lodash-es';

@Injectable({ providedIn: 'root' })
export class ContainerGuard implements CanActivate {
  constructor(private inventoryService: InventoryService) {}

  canActivate(route: ActivatedRouteSnapshot): Promise<boolean> {
    const id = get(route, 'params.id') || get(route, 'parent.params.id');
    return this.inventoryService
      .childAdditionsList(
        { id },
        { query: `serviceType eq container`, pageSize: 1 }
      )
      .then(result => !!result?.data?.length);
  }
}
