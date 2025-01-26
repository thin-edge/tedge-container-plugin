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
        {
          query: `(serviceType eq 'container' or serviceType eq 'container-group') and has(container)`,
          pageSize: 1,
        }
      )
      .then(result => {
        console.log('Verify container:', !!result?.data?.length);
        // !!result?.data?.length
        return !!result?.data?.length;
      });
  }
}
