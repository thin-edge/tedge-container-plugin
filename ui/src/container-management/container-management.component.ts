import { Component, OnInit, ViewEncapsulation } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { ContainerService } from '../shared/container-service';
import { Container } from '../shared/container';

import { IResultList } from '@c8y/client';

@Component({
  selector: 'container-management-tab',
  templateUrl: './container-management.component.html',
  encapsulation: ViewEncapsulation.None,
})
export class ContainerManagementComponent implements OnInit {
  containers: Container[];
  deviceId: string;
  gridOrList: 'interact-list' | 'interact-grid' = 'interact-grid';
  searchText: string = '';
  showContainerGroups: boolean = false;
  isLoading: boolean = true;
  constructor(
    private route: ActivatedRoute,
    private containerservice: ContainerService,
    private router: Router
  ) {}

  ngOnInit(): void {
    this.deviceId = this.route.snapshot.parent.data.contextData['id'];
    this.loadData();
  }

  displayMode(listClass: 'interact-list' | 'interact-grid') {
    this.gridOrList = listClass;
  }

  async loadData() {
    this.containers = await this.containerservice.getContainers(this.deviceId);
  }

  remove(serviceId: string) {
    console.log('Removal of container not supported');
  }

  reload() {
    this.isLoading = true;
    this.searchText = '';
    this.loadData();
  }
}
